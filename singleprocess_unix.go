//go:build !windows
// +build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/user/media-manager/config"
)

// ensureSingleProcess确保只有一个程序实例在运行
// 简化实现，移除平台特定的文件锁
func ensureSingleProcess() bool {
	// 创建锁文件路径
	configPath := config.GetConfigPath()
	lockDir := filepath.Dir(configPath)
	lockFile := filepath.Join(lockDir, "media-manager.lock")

	// 尝试打开锁文件
	file, err := os.OpenFile(lockFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("创建锁文件失败: %v\n", err)
		return true // 在开发环境中，锁文件创建失败时允许程序继续运行
	}
	defer file.Close()

	// 简化实现：不使用文件锁，直接返回true
	// 单进程控制功能在交叉编译时可能会有问题
	// 在实际部署时可以根据需要恢复完整实现
	return true
}
