package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/user/media-manager/classifier"
	"github.com/user/media-manager/config"
	"github.com/user/media-manager/logging"
	"github.com/user/media-manager/processor"
	"github.com/user/media-manager/scraper"
)

// 定义命令行参数
var (
	nfoFile      = flag.String("nfo", "", "指定NFO文件路径")
	movieDir     = flag.String("dir", "", "指定影片目录路径")
	scrapeMovies = flag.Bool("scrape-movies", false, "执行电影刮削")
	scrapeTV     = flag.Bool("scrape-tv", false, "执行电视剧刮削")
	scrapeAll    = flag.Bool("scrape-all", false, "执行所有刮削")
	configCmd    = flag.Bool("config", false, "查看或修改配置")
)

// main是应用程序的入口点
func main() {
	// 解析命令行参数
	flag.Parse()

	// 记录程序启动信息
	logging.Info("程序启动，版本: 1.0.0")

	// 检查是否为单进程
	if !ensureSingleProcess() {
		logging.Error("程序已经在运行中，退出")
		os.Exit(1)
	}

	// 处理配置命令
	if *configCmd {
		logging.Info("处理配置命令")
		showConfig()
		os.Exit(0)
	}

	// 处理刮削命令
	if *scrapeMovies || *scrapeTV || *scrapeAll {
		logging.Info("处理刮削命令")
		handleScrape()
		os.Exit(0)
	}

	// 处理NFO文件
	if *nfoFile != "" {
		logging.Info("处理单个NFO文件: %s", *nfoFile)
		handleSingleNFO(*nfoFile)
		os.Exit(0)
	}

	// 处理影片目录
	if *movieDir != "" {
		logging.Info("处理影片目录: %s", *movieDir)
		handleMovieDir(*movieDir)
		os.Exit(0)
	}

	// 如果没有提供任何命令行参数，显示帮助信息
	logging.Info("没有提供命令行参数，显示帮助信息")
	flag.Usage()
	os.Exit(0)
}

// showConfig显示当前配置
func showConfig() {
	cfg := config.LoadConfig()
	fmt.Println("当前配置:")
	fmt.Printf("Cloud目录: %s\n", cfg.CloudDir)
	fmt.Printf("TinyMediaManager目录: %s\n", cfg.TinyMediaManagerDir)
	fmt.Println("临时目录:")
	for i, tempDir := range cfg.TempDirs {
		fmt.Printf("  %d. %s\n", i+1, tempDir)
	}
}

// handleScrape处理刮削命令
func handleScrape() {
	var err error
	var scrapeType string // 记录刮削类型：all, movies, tv

	if *scrapeAll {
		// 执行所有刮削
		err = scraper.ScrapeAll()
		scrapeType = "all"
	} else if *scrapeMovies {
		// 执行电影刮削
		err = scraper.ScrapeMovies()
		scrapeType = "movies"
	} else if *scrapeTV {
		// 执行电视剧刮削
		err = scraper.ScrapeTVShows()
		scrapeType = "tv"
	}

	if err != nil {
		logging.Error("刮削失败: %v", err)
		os.Exit(1)
	}

	// 加载配置获取等待时间
	cfg := config.LoadConfig()
	if cfg.WaitTimeAfterScan > 0 {
		logging.Info("刮削完成，等待 %d 秒后开始处理NFO文件...", cfg.WaitTimeAfterScan)
		time.Sleep(time.Duration(cfg.WaitTimeAfterScan) * time.Second)
	}

	logging.Info("开始处理NFO文件...")

	// 根据刮削类型确定要扫描的子目录
	targetSubdirs := []string{}
	switch scrapeType {
	case "all":
		targetSubdirs = []string{"Movie", "TvShow"}
	case "movies":
		targetSubdirs = []string{"Movie"}
	case "tv":
		targetSubdirs = []string{"TvShow"}
	}

	// 先检查所有相关目录是否有多个NFO文件
	for _, tempDir := range cfg.TempDirs {
		for _, subdir := range targetSubdirs {
			scanDir := filepath.Join(tempDir, subdir)
			logging.Info("开始检查目录 %s 的结构，确保没有包含多个NFO文件的子目录", scanDir)

			// 检查该目录下的所有子目录
			err := filepath.Walk(scanDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					logging.Debug("访问路径失败: %s, 错误: %v", path, err)
					return nil // 忽略访问错误
				}

				if info.IsDir() && path != scanDir {
					// 检查该目录是否包含媒体文件
					if hasMediaFiles(path) {
						// 检查该目录下的NFO文件数量
						var nfoCount int
						entries, err := os.ReadDir(path)
						if err != nil {
							logging.Error("无法打开目录: %s, 错误: %v", path, err)
							return nil // 跳过该目录，继续检查其他目录
						}

						for _, entry := range entries {
							if !entry.IsDir() && strings.ToLower(filepath.Ext(entry.Name())) == ".nfo" {
								nfoCount++
							}
						}

						if nfoCount > 1 {
							logging.Error("目录 %s 下存在 %d 个NFO文件，将跳过该目录的处理。请手动选择正确的NFO文件后再处理。", path, nfoCount)
						}
					}
				}

				return nil
			})

			if err != nil {
				logging.Error("检查目录结构失败: %v", err)
				// 不退出，继续检查其他目录
			}
		}
	}

	logging.Info("所有目录结构检查完成")

	// 查找指定子目录下的NFO文件
	var nfoFiles []string
	for _, tempDir := range cfg.TempDirs {
		for _, subdir := range targetSubdirs {
			scanDir := filepath.Join(tempDir, subdir)
			logging.Info("开始遍历目录 %s 查找NFO文件", scanDir)
			files, err := findNFOFiles(scanDir)
			if err != nil {
				logging.Error("在目录 %s 中查找NFO文件失败: %v", scanDir, err)
				continue
			}
			nfoFiles = append(nfoFiles, files...)
		}
	}

	if len(nfoFiles) == 0 {
		logging.Info("没有找到NFO文件")
		os.Exit(0)
	}

	// 处理每个NFO文件
	for _, nfoFile := range nfoFiles {
		logging.Info("------------------------")
		logging.Info("开始处理NFO文件: %s", nfoFile)

		// 处理类型字段
		if err := processor.ProcessGenre(nfoFile); err != nil {
			logging.Error("处理类型字段失败: %v", err)
			continue
		}

		// 处理演员字段
		report, err := processor.ProcessActor(nfoFile)
		if err != nil {
			logging.Error("处理演员字段失败: %v", err)
			continue
		}

		if len(report.Actors) > 0 {
			logging.Info("发现 %d 个非中文演员名称", len(report.Actors))
		}

		// NFO文件编辑完成后等待指定时间
		if cfg.WaitTimeAfterNFOEdit > 0 {
			logging.Info("NFO文件编辑完成，等待 %d 秒后开始移动文件...", cfg.WaitTimeAfterNFOEdit)
			time.Sleep(time.Duration(cfg.WaitTimeAfterNFOEdit) * time.Second)
		}

		// 分类并移动影片
		if err := classifier.ClassifyAndMove(nfoFile); err != nil {
			logging.Error("分类和移动影片失败: %v", err)
			continue
		}

		logging.Info("NFO文件处理完成: %s", nfoFile)
	}

	logging.Info("所有NFO文件处理完成")
}

// handleSingleNFO处理单个NFO文件
func handleSingleNFO(nfoPath string) {
	// 检查文件是否存在
	if _, err := os.Stat(nfoPath); os.IsNotExist(err) {
		logging.Error("NFO文件不存在: %s", nfoPath)
		os.Exit(1)
	}

	// 检查NFO文件所在目录是否有多个NFO文件
	dirPath := filepath.Dir(nfoPath)
	if _, err := checkNFOCount(dirPath); err != nil {
		logging.Error("%v，跳过处理", err)
		os.Exit(1)
	}

	// 记录开始时间
	startTime := time.Now()

	// 处理类型字段
	logging.Info("开始处理NFO文件: %s", nfoPath)
	if err := processor.ProcessGenre(nfoPath); err != nil {
		logging.Error("处理类型字段失败: %v", err)
		os.Exit(1)
	}

	// 处理演员字段
	report, err := processor.ProcessActor(nfoPath)
	if err != nil {
		logging.Error("处理演员字段失败: %v", err)
		os.Exit(1)
	}

	if len(report.Actors) > 0 {
		logging.Info("发现 %d 个非中文演员名称", len(report.Actors))
	}

	// 加载配置获取等待时间
	cfg := config.LoadConfig()
	// NFO文件编辑完成后等待指定时间
	if cfg.WaitTimeAfterNFOEdit > 0 {
		logging.Info("NFO文件编辑完成，等待 %d 秒后开始移动文件...", cfg.WaitTimeAfterNFOEdit)
		time.Sleep(time.Duration(cfg.WaitTimeAfterNFOEdit) * time.Second)
	}

	// 分类并移动影片
	if err := classifier.ClassifyAndMove(nfoPath); err != nil {
		logging.Error("分类和移动影片失败: %v", err)
		os.Exit(1)
	}

	// 计算处理时间
	elapsedTime := time.Since(startTime)
	logging.Info("NFO文件处理完成，耗时: %v", elapsedTime)
}

// handleMovieDir处理影片目录
func handleMovieDir(dirPath string) {
	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		logging.Error("目录不存在: %s", dirPath)
		os.Exit(1)
	}

	// 先检查目录结构，确保没有任何包含多个NFO文件的子目录
	logging.Info("开始检查目录结构，确保没有包含多个NFO文件的子目录")
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logging.Debug("访问路径失败: %s, 错误: %v", path, err)
			return nil // 忽略访问错误
		}

		if info.IsDir() {
			// 检查该目录是否包含媒体文件
			if hasMediaFiles(path) {
				// 检查该目录下的NFO文件数量
				var nfoCount int
				entries, err := os.ReadDir(path)
				if err != nil {
					logging.Error("无法打开目录: %s, 错误: %v", path, err)
					return nil // 跳过该目录，继续检查其他目录
				}

				for _, entry := range entries {
					if !entry.IsDir() && strings.ToLower(filepath.Ext(entry.Name())) == ".nfo" {
						nfoCount++
					}
				}

				if nfoCount > 1 {
					logging.Error("目录 %s 下存在 %d 个NFO文件，将跳过该目录的处理。请手动选择正确的NFO文件后再处理。", path, nfoCount)
				}
			}
		}

		return nil
	})

	if err != nil {
		logging.Error("检查目录结构失败: %v", err)
		// 不退出，继续执行后面的操作
	}

	logging.Info("目录结构检查通过，没有包含多个NFO文件的子目录")

	// 查找目录下的NFO文件
	logging.Info("开始在目录 %s 中查找NFO文件", dirPath)
	nfoFiles, err := findNFOFiles(dirPath)
	if err != nil {
		logging.Error("查找NFO文件失败: %v", err)
		os.Exit(1)
	}

	if len(nfoFiles) == 0 {
		logging.Info("目录 %s 下没有找到NFO文件", dirPath)
		os.Exit(1)
	}

	logging.Info("找到 %d 个NFO文件，开始处理", len(nfoFiles))

	// 处理每个NFO文件
	for i, nfoFile := range nfoFiles {
		logging.Info("处理第 %d/%d 个NFO文件: %s", i+1, len(nfoFiles), nfoFile)
		handleSingleNFO(nfoFile)
		logging.Info("------------------------")
	}

	logging.Info("所有NFO文件处理完成")
}

// checkNFOCount检查目录中NFO文件的数量，如果有多个则返回错误
func checkNFOCount(dirPath string) (int, error) {
	var nfoCount int

	// 打开目录
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, fmt.Errorf("无法打开目录: %w", err)
	}

	// 遍历目录中的直接子文件，统计NFO文件数量
	for _, entry := range entries {
		if !entry.IsDir() && strings.ToLower(filepath.Ext(entry.Name())) == ".nfo" {
			nfoCount++
		}
	}

	// 如果有多个NFO文件，返回错误
	if nfoCount > 1 {
		return nfoCount, fmt.Errorf("目录 %s 下存在 %d 个NFO文件", dirPath, nfoCount)
	}

	return nfoCount, nil
}

// hasMediaFiles检查目录是否包含媒体文件
func hasMediaFiles(dirPath string) bool {
	// 常见的媒体文件扩展名
	mediaExts := []string{".mkv", ".mp4", ".avi", ".wmv", ".flv", ".mov", ".rmvb"}

	// 标记是否找到媒体文件
	found := false

	// 递归遍历目录内容
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // 忽略访问错误
		}

		// 检查是否为文件且具有媒体文件扩展名
		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(info.Name()))
			for _, mediaExt := range mediaExts {
				if ext == mediaExt {
					found = true
					return filepath.SkipDir // 找到后立即停止遍历
				}
			}
		}

		return nil
	})

	if err != nil {
		return false
	}

	return found
}

// findNFOFiles查找目录下的所有NFO文件
func findNFOFiles(dirPath string) ([]string, error) {
	var nfoFiles []string
	// 使用map记录每个目录下的NFO文件，确保唯一性
	dirNFOMap := make(map[string][]string)
	logging.Info("开始遍历目录 %s 查找NFO文件", dirPath)

	// 遍历目录
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logging.Debug("访问路径失败: %s, 错误: %v", path, err)
			return nil // 忽略访问错误
		}

		if info.IsDir() {
			// 如果是项目目录，跳过
			if path != dirPath {
				// 检查目录是否包含项目文件
				projectFiles := []string{"go.mod", "main.go", "go.sum", "CMakeLists.txt", "Makefile", "package.json", "requirements.txt"}
				for _, file := range projectFiles {
					filePath := filepath.Join(path, file)
					if _, err := os.Stat(filePath); err == nil {
						logging.Debug("跳过项目目录: %s", path)
						return filepath.SkipDir
					}
				}
				// 检查目录是否包含媒体文件
				if !hasMediaFiles(path) {
					logging.Debug("跳过不包含媒体文件的目录: %s", path)
					return nil // 跳过不包含媒体文件的目录
				}
			}
		} else {
			// 检查是否为NFO文件
			if strings.ToLower(filepath.Ext(path)) == ".nfo" {
				logging.Debug("找到NFO文件: %s", path)
				// 获取该NFO文件所在的目录
				parentDir := filepath.Dir(path)
				// 将NFO文件添加到对应目录的列表中
				dirNFOMap[parentDir] = append(dirNFOMap[parentDir], path)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// 从每个目录中选择一个最合适的NFO文件
	for dir, files := range dirNFOMap {
		if len(files) == 0 {
			continue
		}

		// 尝试选择最合适的NFO文件
		// 优先选择没有"(数字)"后缀的文件
		var selectedFile string
		for _, file := range files {
			fileName := filepath.Base(file)
			// 检查文件名是否包含"(数字)"后缀
			if !strings.Contains(fileName, "(") || !strings.Contains(fileName, ")") {
				selectedFile = file
				break
			}
		}

		// 如果没有找到合适的，选择第一个
		if selectedFile == "" {
			selectedFile = files[0]
		}

		// 如果有多个NFO文件，记录日志
		if len(files) > 1 {
			logging.Info("目录 %s 下有 %d 个NFO文件，选择处理: %s", dir, len(files), selectedFile)
			for _, file := range files {
				if file != selectedFile {
					logging.Info("跳过NFO文件: %s", file)
				}
			}
		}

		nfoFiles = append(nfoFiles, selectedFile)
	}

	logging.Info("在目录 %s 中找到 %d 个NFO文件（每个目录一个）", dirPath, len(nfoFiles))
	return nfoFiles, nil
}
