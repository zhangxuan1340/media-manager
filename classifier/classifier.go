package classifier

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	if _, err := os.Stat(targetMediaPath); err == nil {
		logging.Warning("目标目录已存在同名文件夹 '%s'，跳过移动", targetMediaPath)
		return nil // 跳过移动，但不返回错误
	}

	// 移动文件夹
	if err := moveDirectory(mediaDir, targetMediaPath); err != nil {
		return fmt.Errorf("移动影片失败: %w", err)
	}

	logging.Info("已将影片 '%s' 移动到 '%s'", mediaName, targetDir)

	// 记录媒体信息到数据库
	mediaRecord := &database.MediaRecord{
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
	}

	if err := database.InsertMediaRecord(mediaRecord); err != nil {
		logging.Error("记录媒体信息到数据库失败: %v", err)
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

	// 处理多国家情况
	var hasDomestic bool
	var hasJapanKorea bool

	for _, country := range countries {
		countryLower := strings.ToLower(country)

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
			hasDomestic = true
		}

		// 检查是否为日本或韩国
		if strings.Contains(countryLower, "日本") ||
			strings.Contains(countryLower, "日本国") ||
			strings.Contains(countryLower, "japan") ||
			strings.Contains(countryLower, "韩国") ||
			strings.Contains(countryLower, "大韩民国") ||
			strings.Contains(countryLower, "korea") ||
			strings.Contains(countryLower, "south korea") {
			hasJapanKorea = true
		}
	}

	// 优先处理日本/韩国
	// 解决多国家情况：如果同时包含中国和日韩，优先归类到日韩
	if hasJapanKorea {
		if isTVShow {
			return CategoryJpKrShow, nil
		}
		return CategoryJpKrMovie, nil
	}

	// 其次处理国内
	if hasDomestic {
		if isTVShow {
			return CategoryCnShow, nil
		}
		return CategoryCnMovie, nil
	}

	// 最后处理其他国家
	if isTVShow {
		return CategoryEnShow, nil
	}
	return CategoryEnMovie, nil
}

// moveDirectory处理目录移动，支持跨设备移动
func moveDirectory(src, dst string) error {
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
				if err := moveDirectory(srcPath, dstPath); err != nil {
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
