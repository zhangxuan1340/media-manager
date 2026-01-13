package utils

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetExecutableDir 获取程序执行文件的目录
func GetExecutableDir() (string, error) {
	// 获取程序执行文件的路径
	var exePath string
	var err error

	switch runtime.GOOS {
	case "windows":
		// Windows平台
		exePath, err = os.Executable()
	default:
		// Linux/macOS平台
		exePath, err = os.Executable()
	}

	if err != nil {
		return "", err
	}

	// 处理符号链接，获取实际文件路径
	if runtime.GOOS != "windows" {
		if link, err := os.Readlink(exePath); err == nil && link != "" {
			exePath = link
			// 如果是相对路径，转换为绝对路径
			if !filepath.IsAbs(exePath) {
				exePath = filepath.Join(filepath.Dir(exePath), exePath)
			}
		}
	}

	// 获取目录路径
	exeDir := filepath.Dir(exePath)
	return exeDir, nil
}
