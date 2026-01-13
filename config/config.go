package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/user/media-manager/utils"
)

// Config 支持单个或多个Temp目录
// 当JSON中是字符串时，TempDirs是单元素数组
// 当JSON中是数组时，TempDirs是多元素数组
type Config struct {
	CloudDir             string   `json:"cloud_dir"`
	TinyMediaManagerDir  string   `json:"tiny_media_manager_dir"`
	TempDirs             []string `json:"temp_dir"`
	TMDBApiKey           string   `json:"tmdb_api_key"`             // TMDB API密钥
	UseTMDBOrg           bool     `json:"use_tmdb_org"`             // 是否使用tmdb.org访问API
	WaitTimeAfterScan    int      `json:"wait_time_after_scan"`     // 扫描后等待时间（秒）
	WaitTimeAfterNFOEdit int      `json:"wait_time_after_nfo_edit"` // NFO文件编辑后等待时间（秒）
}

const (
	ConfigDir     = ".media-manager"
	ConfigFile    = "config.json"
	DefaultCloud  = "~/Cloud"
	DefaultTemp   = "~/Temp"
	DefaultTMMDir = "/usr/local/bin" // 默认路径，需要根据实际情况调整
)

func GetConfigPath() string {
	// 1. 首先检查用户当前目录下是否存在config目录（只检查不创建）
	currentDir, err := os.Getwd()
	if err == nil {
		configDir := filepath.Join(currentDir, "config")
		if _, err := os.Stat(configDir); err == nil {
			return filepath.Join(configDir, ConfigFile)
		}
	}

	// 2. 检查程序执行文件所在目录下是否存在config目录（只检查不创建）
	exeDir, err := utils.GetExecutableDir()
	if err == nil {
		configDir := filepath.Join(exeDir, "config")
		if _, err := os.Stat(configDir); err == nil {
			return filepath.Join(configDir, ConfigFile)
		}
	}

	// 3. 如果都不存在，使用用户主目录下的.media-manager目录（不存在则创建）
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("无法获取用户主目录: %v\n", err)
		os.Exit(1)
	}
	configDir := filepath.Join(homeDir, ConfigDir)
	// 确保用户主目录下的配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("无法创建配置目录: %v\n", err)
		os.Exit(1)
	}
	return filepath.Join(configDir, ConfigFile)
}

// configWithFlexibleTemp 用于处理灵活的temp_dir字段（字符串或数组）
type configWithFlexibleTemp struct {
	CloudDir             string          `json:"cloud_dir"`
	TinyMediaManagerDir  string          `json:"tiny_media_manager_dir"`
	TempDir              json.RawMessage `json:"temp_dir"`
	TMDBApiKey           string          `json:"tmdb_api_key"`
	UseTMDBOrg           bool            `json:"use_tmdb_org"`
	WaitTimeAfterScan    int             `json:"wait_time_after_scan"`
	WaitTimeAfterNFOEdit int             `json:"wait_time_after_nfo_edit"`
}

func LoadConfig() *Config {
	configPath := GetConfigPath()

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 创建默认配置
		config := createDefaultConfig()
		// 保存默认配置
		SaveConfig(config)
		fmt.Printf("已创建默认配置文件: %s\n", configPath)
		return config
	}

	// 读取配置文件
	file, err := os.Open(configPath)
	if err != nil {
		fmt.Printf("无法打开配置文件: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 解析JSON到临时结构体
	var tempConfig configWithFlexibleTemp
	if err := json.NewDecoder(file).Decode(&tempConfig); err != nil {
		fmt.Printf("无法解析配置文件: %v\n", err)
		os.Exit(1)
	}

	// 处理灵活的TempDir字段
	var config Config
	config.CloudDir = tempConfig.CloudDir
	config.TinyMediaManagerDir = tempConfig.TinyMediaManagerDir
	config.TMDBApiKey = tempConfig.TMDBApiKey
	config.UseTMDBOrg = tempConfig.UseTMDBOrg
	config.WaitTimeAfterScan = tempConfig.WaitTimeAfterScan
	config.WaitTimeAfterNFOEdit = tempConfig.WaitTimeAfterNFOEdit

	// 解析TempDir字段（可能是字符串或数组）
	if tempConfig.TempDir[0] == '[' {
		// 是数组
		if err := json.Unmarshal(tempConfig.TempDir, &config.TempDirs); err != nil {
			fmt.Printf("无法解析temp_dir数组: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 是字符串
		var tempDir string
		if err := json.Unmarshal(tempConfig.TempDir, &tempDir); err != nil {
			fmt.Printf("无法解析temp_dir字符串: %v\n", err)
			os.Exit(1)
		}
		config.TempDirs = []string{tempDir}
	}

	// 替换路径中的 ~ 为用户主目录
	config.CloudDir = expandHomePath(config.CloudDir)
	config.TinyMediaManagerDir = expandHomePath(config.TinyMediaManagerDir)

	// 处理所有TempDirs
	validTempDirs := []string{}
	for _, tempDir := range config.TempDirs {
		// 替换~为用户主目录
		expandedTempDir := expandHomePath(tempDir)

		// 检查目录是否存在
		if _, err := os.Stat(expandedTempDir); os.IsNotExist(err) {
			fmt.Printf("警告: 配置文件中指定的Temp目录不存在: %s\n", expandedTempDir)
			continue
		}

		validTempDirs = append(validTempDirs, expandedTempDir)
	}

	// 如果没有有效Temp目录，使用空切片
	if len(validTempDirs) == 0 {
		fmt.Printf("警告: 没有找到有效Temp目录\n")
		// 不创建默认目录，返回空切片
		// 这确保了即使没有Temp目录，程序也能继续运行
		validTempDirs = []string{}
	}

	config.TempDirs = validTempDirs

	return &config
}

func SaveConfig(config *Config) {
	configPath := GetConfigPath()
	configDir := filepath.Dir(configPath)

	// 确保配置目录存在
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("无法创建配置目录: %v\n", err)
		os.Exit(1)
	}

	// 创建配置文件
	file, err := os.Create(configPath)
	if err != nil {
		fmt.Printf("无法创建配置文件: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 序列化JSON
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		fmt.Printf("无法保存配置文件: %v\n", err)
		os.Exit(1)
	}
}

func createDefaultConfig() *Config {
	// 根据操作系统设置默认路径
	tmmDir := DefaultTMMDir
	switch runtime.GOOS {
	case "windows":
		tmmDir = "C:/Program Files/tinyMediaManager"
		// 其他Windows特定的设置
	case "darwin":
		tmmDir = "/Applications/tinyMediaManager.app/Contents/MacOS"
		// 其他macOS特定的设置
	}

	return &Config{
		CloudDir:             DefaultCloud,
		TinyMediaManagerDir:  tmmDir,
		TempDirs:             []string{DefaultTemp},
		TMDBApiKey:           "",    // 默认为空，需要用户手动配置
		UseTMDBOrg:           false, // 默认不使用tmdb.org
		WaitTimeAfterScan:    30,    // 默认等待时间30秒
		WaitTimeAfterNFOEdit: 10,    // 默认NFO文件编辑后等待时间10秒
	}
}

// expandHomePath 替换路径中的 ~ 为用户主目录
func expandHomePath(path string) string {
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path // 出错时返回原路径
		}
		return filepath.Join(homeDir, path[1:])
	}
	return path
}
