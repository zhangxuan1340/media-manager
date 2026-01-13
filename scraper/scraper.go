package scraper

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/user/media-manager/config"
	"github.com/user/media-manager/logging"
)

// ScrapeMovies执行电影刮削命令
func ScrapeMovies() error {
	cfg := config.LoadConfig()

	// 检查tinyMediaManager可执行文件是否存在
	tmmPath := getTMMExecutablePath(cfg)
	if _, err := os.Stat(tmmPath); os.IsNotExist(err) {
		return fmt.Errorf("tinyMediaManager可执行文件不存在: %s\n请检查配置文件中的TinyMediaManagerDir路径是否正确", tmmPath)
	}

	// 使用第一个有效的TempDir作为工作目录
	if len(cfg.TempDirs) == 0 {
		return fmt.Errorf("没有有效的临时目录可用")
	}

	// 构建命令
	cmd := exec.Command(tmmPath, "movie", "-u", "-n", "-r")
	cmd.Dir = cfg.TempDirs[0] // 设置工作目录为第一个临时目录

	// 设置输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logging.Info("开始刮削电影...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("刮削电影失败: %w", err)
	}

	logging.Info("电影刮削完成")
	return nil
}

// ScrapeTVShows执行电视剧刮削命令
func ScrapeTVShows() error {
	cfg := config.LoadConfig()

	// 检查tinyMediaManager可执行文件是否存在
	tmmPath := getTMMExecutablePath(cfg)
	if _, err := os.Stat(tmmPath); os.IsNotExist(err) {
		return fmt.Errorf("tinyMediaManager可执行文件不存在: %s\n请检查配置文件中的TinyMediaManagerDir路径是否正确", tmmPath)
	}

	// 使用第一个有效的TempDir作为工作目录
	if len(cfg.TempDirs) == 0 {
		return fmt.Errorf("没有有效的临时目录可用")
	}

	// 构建命令
	cmd := exec.Command(tmmPath, "tvshow", "-u", "-n", "-r")
	cmd.Dir = cfg.TempDirs[0] // 设置工作目录为第一个临时目录

	// 设置输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logging.Info("开始刮削电视剧...")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("刮削电视剧失败: %w", err)
	}

	logging.Info("电视剧刮削完成")
	return nil
}

// ScrapeAll执行所有刮削命令
func ScrapeAll() error {
	// 执行电影刮削
	if err := ScrapeMovies(); err != nil {
		return err
	}

	// 执行电视剧刮削
	if err := ScrapeTVShows(); err != nil {
		return err
	}

	return nil
}

// getTMMExecutablePath获取tinyMediaManager可执行文件的完整路径
func getTMMExecutablePath(cfg *config.Config) string {
	// 根据操作系统确定可执行文件名
	executableName := "tinymediamanager"

	// 检查是否为macOS应用程序包
	if strings.Contains(cfg.TinyMediaManagerDir, ".app") {
		return filepath.Join(cfg.TinyMediaManagerDir, executableName)
	}

	// 检查是否为Windows系统
	if strings.Contains(cfg.TinyMediaManagerDir, "Program Files") {
		executableName += ".exe"
	}

	// 检查Linux系统上的可执行文件（大小写敏感）
	// 先尝试首字母大写的版本
	linuxExecutableName := "tinyMediaManager"
	linuxPath := filepath.Join(cfg.TinyMediaManagerDir, linuxExecutableName)
	if _, err := os.Stat(linuxPath); err == nil {
		return linuxPath
	}

	// 如果首字母大写的版本不存在，再尝试全小写的版本
	return filepath.Join(cfg.TinyMediaManagerDir, executableName)
}
