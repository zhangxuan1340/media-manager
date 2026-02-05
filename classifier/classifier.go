package classifier

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/user/media-manager/config"
	"github.com/user/media-manager/database"
	"github.com/user/media-manager/logging"
	"github.com/user/media-manager/parser"
	"github.com/user/media-manager/tmdb"
	"github.com/user/media-manager/utils"
)

// Category定义分类常量
const (
	CategoryCnMovie   = "CnMovie"    // 国内电影
	CategoryCnShow    = "CnShow"     // 国内电视剧
	CategoryEnMovie   = "EnMovie"    // 非中日韩及港澳台地区的电影
	CategoryEnShow    = "EnShow"     // 非中日韩及港澳台地区的电视剧
	CategoryJpKrShow  = "Jp&KrShow"  // 日本和韩国的电视剧
	CategoryJpKrMovie = "Jp&KrMovie" // 日本和韩国的电影
	CategoryDmMovie   = "DmMovie"    // 动漫电影
	CategoryDmShow    = "DmShow"     // 动漫剧集
	CategoryJlShow    = "JlShow"     // 纪录片
	CategoryXSShow    = "XSShow"     // 综艺节目
)

// isProjectDirectory检查目录是否为项目目录
func isProjectDirectory(dirPath string) bool {
	// 检查目录是否包含项目标志性文件
	projectFiles := []string{
		"go.mod", "main.go", "go.sum", // Go项目
		"CMakeLists.txt", "Makefile", // C++/C项目
		"package.json",            // Node.js项目
		"requirements.txt",        // Python项目
		"README.md", "README.txt", // 文档文件
		".git", // Git版本控制
	}
	for _, file := range projectFiles {
		filePath := filepath.Join(dirPath, file)
		if _, err := os.Stat(filePath); err == nil {
			return true
		}
	}
	return false
}

// ClassifyAndMove根据国家/地区和类型分类并移动影片
func ClassifyAndMove(nfoPath string) error {
	// 解析NFO文件
	nfo, err := parser.ParseNFO(nfoPath)
	if err != nil {
		return fmt.Errorf("分类时解析NFO文件失败: %w", err)
	}

	// 加载配置
	cfg := config.LoadConfig()

	// 获取影片目录
	mediaDir := filepath.Dir(nfoPath)
	mediaName := filepath.Base(mediaDir)

	// 检查NFO文件所在目录是否有多个NFO文件
	var nfoCount int
	entries, err := os.ReadDir(mediaDir)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.ToLower(filepath.Ext(entry.Name())) == ".nfo" {
			nfoCount++
		}
	}

	if nfoCount > 1 {
		logging.Error("目录 %s 下存在 %d 个NFO文件，跳过移动。请手动选择正确的NFO文件后再处理。", mediaDir, nfoCount)
		return nil // 跳过移动，不返回错误
	}

	// 检查NFO文件是否包含足够信息
	if !isNFOResolved(nfo) {
		logging.Info("NFO文件信息不完整（可能未正确刮削），跳过移动: %s", nfoPath)
		return nil
	}

	// 确定分类
	isTVShow := nfo.IsTVShow()

	// 使用TMDB API获取原始产地信息（如果有TMDbID）
	countries := nfo.Country
	if nfo.TMDbID != "" {
		cfg := config.LoadConfig()
		if cfg.TMDBApiKey != "" {
			// 尝试从TMDB获取制作国家信息
			tmdbCountries, err := tmdb.GetProductionCountries(nfo.TMDbID, isTVShow)
			if err != nil {
				logging.Warning("从TMDB获取制作国家信息失败: %v，将使用NFO文件中的国家信息", err)
			} else {
				countries = tmdbCountries
				logging.Info("从TMDB获取到的制作国家: %v", countries)
			}
		}
	}

	// 检查国家信息是否为空，如果为空则跳过移动
	if len(countries) == 0 {
		logging.Warning("没有获取到有效的国家信息，跳过移动: %s", mediaDir)
		return nil
	}

	category, err := DetermineCategory(countries, isTVShow, nfo.Genres)
	if err != nil {
		return fmt.Errorf("确定分类失败: %w", err)
	}

	// 检查是否为项目目录
	if isProjectDirectory(mediaDir) {
		logging.Info("跳过移动项目目录: %s", mediaDir)
		return nil
	}

	// 检查标题是否为简体中文
	if !utils.IsSimplifiedChinese(nfo.Title) {
		logging.Info("标题 '%s' 不是简体中文，跳过移动", nfo.Title)
		return nil
	}

	// 检查所有类型是否为简体中文
	for _, genre := range nfo.Genres {
		if !utils.IsSimplifiedChinese(genre) {
			logging.Info("类型 '%s' 不是简体中文，跳过移动", genre)
			return nil
		}
	}

	// 目标目录路径
	targetDir := filepath.Join(cfg.CloudDir, category)

	// 确保目标目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 检查目标目录是否已存在同名文件夹
	targetMediaPath := filepath.Join(targetDir, mediaName)

	// 从文件名中提取分辨率信息 - 在移动前处理
	resolution := extractResolutionFromFileName(filepath.Base(nfoPath))

	// 对于电视剧合并季数的情况，需要先获取现有记录 - 在移动前处理
	var mediaRecord *database.MediaRecord

	// 尝试获取现有媒体记录 - 在移动前处理
	if isTVShow {
		var err error
		mediaRecord, err = getExistingTVShowRecord(nfo.Title, nfo.Year)
		if err != nil {
			logging.Error("获取现有电视剧记录失败: %v", err)
		}
	}

	// 如果没有现有记录或不是电视剧，创建新记录 - 在移动前处理
	if mediaRecord == nil {
		mediaRecord = &database.MediaRecord{
			FileName:      filepath.Base(nfoPath),
			Title:         nfo.Title,
			OriginalTitle: nfo.OriginalTitle,
			Year:          nfo.Year,
			Country:       strings.Join(countries, ", "), // 使用获取到的国家信息（可能来自TMDB）
			Genres:        strings.Join(nfo.Genres, ", "),
			Actors:        formatActors(nfo.Actors),
			Category:      category,
			SourcePath:    mediaDir,
			TargetPath:    targetMediaPath,
			ProcessedAt:   time.Now(),
			Runtime:       nfo.Runtime,
			Plot:          nfo.Plot,
			IMDbID:        nfo.IMDbID,
			TMDbID:        nfo.TMDbID,
			Season:        nfo.Season,
			Episode:       nfo.Episode,
			Director:      nfo.Director,
			Writer:        nfo.Writer,
			Rating:        nfo.Rating,
			Resolution:    resolution,
			IsComplete:    false, // 默认标记为不完整，后续会更新
		}
	} else {
		// 更新现有记录的信息 - 在移动前处理
		mediaRecord.FileName = filepath.Base(nfoPath)
		mediaRecord.OriginalTitle = nfo.OriginalTitle
		mediaRecord.Year = nfo.Year
		mediaRecord.Country = strings.Join(countries, ", ")
		mediaRecord.Genres = strings.Join(nfo.Genres, ", ")
		mediaRecord.Actors = formatActors(nfo.Actors)
		mediaRecord.Category = category
		mediaRecord.TargetPath = targetMediaPath
		mediaRecord.Runtime = nfo.Runtime
		mediaRecord.Plot = nfo.Plot
		mediaRecord.IMDbID = nfo.IMDbID
		mediaRecord.TMDbID = nfo.TMDbID
		mediaRecord.Director = nfo.Director
		mediaRecord.Writer = nfo.Writer
		mediaRecord.Rating = nfo.Rating
		mediaRecord.Resolution = resolution
	}

	// 目标目录已存在同名文件夹
	if _, err := os.Stat(targetMediaPath); err == nil {
		// 只有电视剧才进行季数检测和合并
		if isTVShow {
			// 目标目录已存在，检查是否有新的季数
			hasNew, seasonsToAdd, err := HasNewSeasons(mediaDir, targetMediaPath)
			if err != nil {
				logging.Error("检查新季数失败: %v，跳过移动", err)
				return nil // 跳过移动，但不返回错误
			}

			if hasNew {
				// 存在新的季数，允许移动并合并
				logging.Info("目标目录已存在，但检测到新的季数 %v，将合并到目标目录", seasonsToAdd)

				// 遍历源目录下的所有内容
				entries, err := os.ReadDir(mediaDir)
				if err != nil {
					return fmt.Errorf("读取源目录失败: %w", err)
				}

				for _, entry := range entries {
					srcPath := filepath.Join(mediaDir, entry.Name())
					dstPath := filepath.Join(targetMediaPath, entry.Name())

					// 检查目标路径是否已存在
					if _, err := os.Stat(dstPath); os.IsNotExist(err) {
						// 目标路径不存在，直接移动
						if entry.IsDir() {
							if err := MoveDirectory(srcPath, dstPath); err != nil {
								logging.Error("移动目录失败: %v，跳过该目录", err)
								continue
							}
						} else {
							if err := os.Rename(srcPath, dstPath); err != nil {
								logging.Error("移动文件失败: %v，跳过该文件", err)
								continue
							}
						}
						logging.Info("已将 '%s' 合并到目标目录", entry.Name())
					} else {
						// 检查是否为季数目录，且季数不在现有目录中
						seasonNum := GetSeasonNumberFromDirName(entry.Name())
						if seasonNum > 0 {
							// 检查该季数是否已存在于目标目录
							existingSeasons, _ := GetExistingSeasons(targetMediaPath)
							found := false
							for _, existingSeason := range existingSeasons {
								if existingSeason == seasonNum {
									found = true
									break
								}
							}
							if !found {
								// 该季数不存在，允许移动
								if err := MoveDirectory(srcPath, dstPath); err != nil {
									logging.Error("移动季数目录失败: %v，跳过该目录", err)
									continue
								}
								logging.Info("已将季数 %d 合并到目标目录", seasonNum)
							} else {
								logging.Warning("季数 %d 已存在于目标目录，跳过移动", seasonNum)
							}
						} else {
							logging.Warning("目标目录已存在 '%s'，跳过移动", entry.Name())
						}
					}
				}

				// 删除源目录（如果为空）
				if err := os.RemoveAll(mediaDir); err != nil {
					logging.Warning("删除源目录失败: %v", err)
				} else {
					logging.Info("已删除空的源目录: %s", mediaDir)
				}

				logging.Info("已将影片 '%s' 的新季数合并到目标目录 '%s'", mediaName, targetDir)
			} else {
				logging.Warning("目标目录已存在同名文件夹 '%s'，且没有检测到新的季数，跳过移动", targetMediaPath)
				return nil // 跳过移动，但不返回错误
			}
		} else {
			// 电影直接跳过移动
			logging.Warning("目标目录已存在同名文件夹 '%s'，跳过移动", targetMediaPath)
			return nil // 跳过移动，但不返回错误
		}
	} else {
		// 目标目录不存在，直接移动整个文件夹
		// 移动文件夹
		if err := MoveDirectory(mediaDir, targetMediaPath); err != nil {
			return fmt.Errorf("移动影片失败: %w", err)
		}

		logging.Info("已将影片 '%s' 移动到 '%s'", mediaName, targetDir)
	}

	// 记录媒体信息到数据库 - 在移动后执行，确保路径正确
	if err := database.InsertOrUpdateMediaRecord(mediaRecord); err != nil {
		logging.Error("记录媒体信息到数据库失败: %v", err)
	}

	// 如果是电视剧，检测缺失的季和剧集 - 在移动后执行，确保路径正确
	if isTVShow && nfo.TMDbID != "" {
		if err := DetectMissingSeasonsAndEpisodes(mediaRecord); err != nil {
			logging.Error("检测缺失季和剧集失败: %v", err)
		}
	}

	// 如果是电视剧，检查并报告季数状态 - 在移动后执行，确保路径正确
	if isTVShow && nfo.TMDbID != "" {
		if err := ReportSeasonStatus(nfo.Title, nfo.TMDbID, targetMediaPath); err != nil {
			logging.Error("报告剧集季数状态失败: %v", err)
		}
	}

	return nil
}

// isNFOResolved 检查NFO文件是否包含足够信息（是否已正确刮削）
func isNFOResolved(nfo *parser.NFO) bool {
	// 检查基本信息
	if nfo.Title == "" {
		return false
	}

	// 检查关键信息（至少有一个即可认为已刮削）
	if len(nfo.Genres) > 0 ||
		len(nfo.Country) > 0 ||
		nfo.IMDbID != "" ||
		nfo.TMDbID != "" ||
		nfo.Plot != "" ||
		len(nfo.Actors) > 0 {
		return true
	}

	// 特殊处理：检查标题是否包含中文和年份
	// 例如："笑傲江湖 (1990)" 可能已刮削但某些字段为空
	if utils.IsSimplifiedChinese(nfo.Title) && nfo.Year != "" {
		return true
	}

	return false
}

// DetermineCategory根据国家/地区、类型和 genres 确定分类
func DetermineCategory(countries []string, isTVShow bool, genres []string) (string, error) {
	// 检查是否为纪录片
	for _, genre := range genres {
		if strings.Contains(strings.ToLower(genre), "纪录片") || strings.Contains(strings.ToLower(genre), "documentary") {
			return CategoryJlShow, nil
		}
	}

	// 检查是否为综艺节目
	for _, genre := range genres {
		// 综艺相关关键词列表
		varietyKeywords := []string{
			"综艺节目", "variety",
			"真人秀", "reality",
			"真人节目", "reality show",
			"脱口秀", "talk show",
			"游戏节目", "game-show",
			"竞赛节目", "competition",
			"选秀节目", "talent show",
		}

		// 转换为小写进行匹配
		genreLower := strings.ToLower(genre)

		// 检查是否包含任何综艺关键词
		for _, keyword := range varietyKeywords {
			if strings.Contains(genreLower, keyword) {
				return CategoryXSShow, nil
			}
		}
	}

	// 检查是否为动漫
	isAnime := false
	for _, genre := range genres {
		if strings.Contains(strings.ToLower(genre), "动漫") || strings.Contains(strings.ToLower(genre), "动画") || strings.Contains(strings.ToLower(genre), "animation") {
			isAnime = true
			break
		}
	}

	if isAnime {
		if isTVShow {
			return CategoryDmShow, nil
		}
		return CategoryDmMovie, nil
	}

	// 处理多国家情况：按照国家顺序优先判断第一个国家
	for _, country := range countries {
		countryLower := strings.ToLower(country)

		// 检查是否为日本或韩国
		if strings.Contains(countryLower, "日本") ||
			strings.Contains(countryLower, "日本国") ||
			strings.Contains(countryLower, "japan") ||
			strings.Contains(countryLower, "韩国") ||
			strings.Contains(countryLower, "大韩民国") ||
			strings.Contains(countryLower, "korea") ||
			strings.Contains(countryLower, "south korea") {
			if isTVShow {
				return CategoryJpKrShow, nil
			}
			return CategoryJpKrMovie, nil
		}

		// 检查是否为国内（中国大陆、香港、台湾）
		if strings.Contains(countryLower, "中国大陆") ||
			strings.Contains(countryLower, "中国香港") ||
			strings.Contains(countryLower, "香港特别行政区") ||
			strings.Contains(countryLower, "中国台湾") ||
			strings.Contains(countryLower, "台湾地区") ||
			strings.Contains(countryLower, "香港") ||
			strings.Contains(countryLower, "台湾") ||
			strings.Contains(countryLower, "中国") ||
			strings.Contains(countryLower, "中华人民共和国") ||
			strings.Contains(countryLower, "china") ||
			strings.Contains(countryLower, "hong kong") ||
			strings.Contains(countryLower, "taiwan") {
			if isTVShow {
				return CategoryCnShow, nil
			}
			return CategoryCnMovie, nil
		}

		// 如果不是日韩或中国，归类为其他国家
		if isTVShow {
			return CategoryEnShow, nil
		}
		return CategoryEnMovie, nil
	}

	// 没有国家信息，返回错误
	return "", fmt.Errorf("没有有效的国家信息")
}

// MoveDirectory处理目录移动，支持跨设备移动
func MoveDirectory(src, dst string) error {
	// 首先尝试使用os.Rename，如果成功则直接返回
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		// 如果不是因为文件不存在而失败，可能是跨设备移动
		// 此时需要复制目录然后删除源目录
		fmt.Printf("跨设备移动，使用复制模式: %s -> %s\n", src, dst)

		// 创建目标目录
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}

		// 遍历源目录并复制所有文件和子目录
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			srcPath := filepath.Join(src, entry.Name())
			dstPath := filepath.Join(dst, entry.Name())

			if entry.IsDir() {
				// 递归复制子目录
				if err := MoveDirectory(srcPath, dstPath); err != nil {
					return err
				}
			} else {
				// 复制文件
				if err := copyFile(srcPath, dstPath); err != nil {
					return err
				}
			}
		}

		// 复制完成后删除源目录
		if err := os.RemoveAll(src); err != nil {
			return err
		}

		return nil
	}

	// 如果是因为文件不存在而失败，返回错误
	return fmt.Errorf("源目录不存在: %s", src)
}

// copyFile复制单个文件
func copyFile(src, dst string) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// 复制文件内容
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// 复制文件权限
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}

// formatActors格式化演员列表为字符串
func formatActors(actors []parser.Actor) string {
	var names []string
	for _, actor := range actors {
		names = append(names, actor.Name)
	}
	return strings.Join(names, ", ")
}

// GetSeasonNumberFromDirName 从目录名中提取季数
func GetSeasonNumberFromDirName(dirName string) int {
	// 支持多种季数格式，如 "Season 1", "第1季", "S01", "S1"
	var seasonNumber int

	// 正则表达式匹配季数
	regexPatterns := []string{
		`(?i)season\s*(\d+)`, // Season 1, season 2
		`(?i)第(\d+)季`,        // 第1季, 第2季
		`(?i)S(\d+)`,         // S01, S1
	}

	for _, pattern := range regexPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(dirName)
		if len(matches) > 1 {
			if n, err := strconv.Atoi(matches[1]); err == nil {
				seasonNumber = n
				break
			}
		}
	}

	return seasonNumber
}

// GetExistingSeasons 获取目标目录中已存在的季数
func GetExistingSeasons(targetMediaPath string) ([]int, error) {
	var existingSeasons []int

	// 检查目标目录是否存在
	if _, err := os.Stat(targetMediaPath); os.IsNotExist(err) {
		return existingSeasons, nil
	}

	// 遍历目标目录下的子目录
	entries, err := os.ReadDir(targetMediaPath)
	if err != nil {
		return nil, fmt.Errorf("读取目标目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			seasonNumber := GetSeasonNumberFromDirName(entry.Name())
			if seasonNumber > 0 {
				existingSeasons = append(existingSeasons, seasonNumber)
			}
		}
	}

	return existingSeasons, nil
}

// GetNewSeasons 获取源目录中包含的季数
func GetNewSeasons(mediaDir string) ([]int, error) {
	var newSeasons []int

	// 遍历源目录下的子目录
	entries, err := os.ReadDir(mediaDir)
	if err != nil {
		return nil, fmt.Errorf("读取源目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			seasonNumber := GetSeasonNumberFromDirName(entry.Name())
			if seasonNumber > 0 {
				newSeasons = append(newSeasons, seasonNumber)
			}
		}
	}

	// 如果源目录下没有季数子目录，检查当前目录的季数
	if len(newSeasons) == 0 {
		seasonNumber := GetSeasonNumberFromDirName(filepath.Base(mediaDir))
		if seasonNumber > 0 {
			newSeasons = append(newSeasons, seasonNumber)
		}
	}

	return newSeasons, nil
}

// HasNewSeasons 检查源目录中是否包含目标目录中不存在的季数
func HasNewSeasons(mediaDir string, targetMediaPath string) (bool, []int, error) {
	// 获取目标目录中已存在的季数
	existingSeasons, err := GetExistingSeasons(targetMediaPath)
	if err != nil {
		return false, nil, err
	}

	// 获取源目录中包含的季数
	newSeasons, err := GetNewSeasons(mediaDir)
	if err != nil {
		return false, nil, err
	}

	// 检查是否有新季数
	var seasonsToAdd []int
	for _, newSeason := range newSeasons {
		found := false
		for _, existingSeason := range existingSeasons {
			if newSeason == existingSeason {
				found = true
				break
			}
		}
		if !found {
			seasonsToAdd = append(seasonsToAdd, newSeason)
		}
	}

	return len(seasonsToAdd) > 0, seasonsToAdd, nil
}

// checkSeasonCompleteness 检查剧集季数是否完整
func checkSeasonCompleteness(tmdbID string, existingSeasons []int) (bool, []int, int, error) {
	// 获取剧集总季数
	totalSeasons, err := tmdb.GetTVShowSeasons(tmdbID)
	if err != nil {
		return false, nil, 0, err
	}

	// 检查缺失的季数
	var missingSeasons []int
	for i := 1; i <= totalSeasons; i++ {
		found := false
		for _, season := range existingSeasons {
			if season == i {
				found = true
				break
			}
		}
		if !found {
			missingSeasons = append(missingSeasons, i)
		}
	}

	return len(missingSeasons) == 0, missingSeasons, totalSeasons, nil
}

// extractResolutionFromFileName 从文件名中提取分辨率信息
func extractResolutionFromFileName(fileName string) string {
	// 支持的分辨率格式：1080p, 720p, 4K, 2160p, 1440p等
	resolutionPatterns := []string{
		`(?i)(\d{3,4}p)`, // 1080p, 720p, 2160p, 1440p
		`(?i)(\d{3,4}i)`, // 1080i, 720i
		`(?i)(4k|8k)`,    // 4K, 8K
	}

	for _, pattern := range resolutionPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(fileName)
		if len(matches) > 1 {
			return strings.ToUpper(matches[1])
		}
	}

	return ""
}

// DetectMissingSeasonsAndEpisodes 检测缺失的季和剧集（公共函数）
func DetectMissingSeasonsAndEpisodes(mediaRecord *database.MediaRecord) error {
	if mediaRecord.TMDbID == "" {
		return nil
	}

	// 获取剧集总季数
	totalSeasons, err := tmdb.GetTVShowSeasons(mediaRecord.TMDbID)
	if err != nil {
		return fmt.Errorf("获取剧集总季数失败: %w", err)
	}

	// 获取已存在的季数
	existingSeasons, err := GetExistingSeasons(mediaRecord.TargetPath)
	if err != nil {
		return fmt.Errorf("获取已存在季数失败: %w", err)
	}

	// 记录缺失的季数
	for i := 1; i <= totalSeasons; i++ {
		found := false
		for _, season := range existingSeasons {
			if season == i {
				found = true
				break
			}
		}
		if !found {
			// 记录缺失的季
			missingSeason := &database.MissingSeason{
				MediaID:       mediaRecord.ID,
				Title:         mediaRecord.Title,
				OriginalTitle: mediaRecord.OriginalTitle,
				TMDbID:        mediaRecord.TMDbID,
				Season:        i,
			}
			if err := database.InsertMissingSeason(missingSeason); err != nil {
				logging.Error("记录缺失季失败: %v", err)
			}
		}
	}

	// 检查是否完整
	isComplete := len(existingSeasons) == totalSeasons

	// 更新媒体记录的完整性状态
	mediaRecord.IsComplete = isComplete
	if err := database.InsertOrUpdateMediaRecord(mediaRecord); err != nil {
		logging.Error("更新媒体记录完整性状态失败: %v", err)
	}

	return nil
}

// getExistingTVShowRecord 根据标题和年份获取现有电视剧记录
func getExistingTVShowRecord(title, year string) (*database.MediaRecord, error) {
	// 获取所有媒体记录，然后筛选出匹配的电视剧记录
	mediaRecords, err := database.GetMediaRecords(map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	for _, record := range mediaRecords {
		// 检查是否为电视剧且标题和年份匹配
		if strings.Contains(record.Category, "Show") && record.Title == title && record.Year == year {
			return &record, nil
		}
	}

	return nil, nil
}

// ReportSeasonStatus 报告剧集季数状态
func ReportSeasonStatus(title string, tmdbID string, targetMediaPath string) error {
	// 获取已存在的季数
	existingSeasons, err := GetExistingSeasons(targetMediaPath)
	if err != nil {
		return err
	}

	// 检查季数完整性
	isComplete, missingSeasons, totalSeasons, err := checkSeasonCompleteness(tmdbID, existingSeasons)
	if err != nil {
		logging.Warning("无法检查剧集 '%s' 的季数完整性: %v", title, err)
		return nil
	}

	logging.Info("剧集 '%s' 季数状态报告:", title)
	logging.Info("  - 总季数: %d", totalSeasons)
	logging.Info("  - 已收集季数: %v", existingSeasons)

	if isComplete {
		logging.Info("  - 状态: 完整")
	} else {
		logging.Info("  - 状态: 缺失季数 %v", missingSeasons)
	}

	return nil
}
