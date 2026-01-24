# Media Manager (媒体管理器)

一个功能强大的媒体文件管理工具，用于自动处理、分类和组织电影和电视剧文件。

## 功能特性

### 🎬 核心功能
- **NFO文件处理**：自动解析和处理媒体文件的NFO元数据文件
- **元数据刮削**：支持从TinyMediaManager和TMDB获取电影和电视剧的元数据
- **分类移动**：自动将处理好的媒体文件分类并移动到指定目录
- **多平台支持**：兼容Linux、Windows和macOS系统
- **单进程运行**：确保程序在任何时刻只有一个实例运行，避免冲突

### 🔧 高级特性
- **批量处理**：支持批量处理单个文件、单个目录或多个目录
- **灵活配置**：支持自定义云存储目录、临时目录和等待时间
- **日志记录**：详细的日志记录，便于问题排查
- **演员和类型处理**：自动处理和标准化演员名称和类型信息

## 目录结构

```
├── build/                  # 编译输出目录
├── classifier/             # 媒体文件分类模块
│   └── classifier.go       # 分类和移动逻辑
├── config/                 # 配置管理模块
│   ├── config.go           # 配置加载和保存
│   ├── config.json         # 配置文件
│   └── media-manager.lock  # 单进程锁文件
├── database/               # 数据库相关模块
│   └── database.go         # 数据库操作
├── logging/                # 日志模块
│   └── logging.go          # 日志功能实现
├── parser/                 # NFO文件解析模块
│   └── nfo.go              # NFO文件解析逻辑
├── processor/              # 元数据处理模块
│   ├── actor.go            # 演员信息处理
│   └── genre.go            # 类型信息处理
├── scraper/                # 元数据刮削模块
│   └── scraper.go          # 刮削功能实现
├── tmdb/                   # TMDB API模块
│   └── tmdb.go             # TMDB API交互
├── utils/                  # 工具函数模块
│   ├── chinese.go          # 中文处理工具
│   └── path.go             # 路径处理工具
├── Makefile                # 编译脚本
├── go.mod                  # Go模块依赖
├── go.sum                  # Go模块校验
├── main.go                 # 程序入口
├── singleprocess_unix.go   # Unix系统单进程实现
└── singleprocess_windows.go # Windows系统单进程实现
```

## 配置说明

### 配置文件结构

配置文件位于以下位置之一（优先级从高到低）：
1. 当前执行目录下的 `config/config.json`
2. 程序执行文件所在目录下的 `config/config.json`
3. 用户主目录下的 `.media-manager/config.json`

配置文件格式：

```json
{
  "cloud_dir": "~/Cloud", # 云存储目录路径，处理后的媒体文件会被移动到这里
  "tiny_media_manager_dir": "/vol1/1000/config/tinymediaManager", # TinyMediaManager的安装目录
  "temp_dir": "~/Temp", # 临时目录路径，用于存储中间文件,可以多个目录
  "tmdb_api_key": "your_tmdb_api_key", # TMDB API密钥，用于获取元数据
  "wait_time_after_scan": 30, # 扫描后等待时间（秒），确保所有文件都已准备好
  "wait_time_after_nfo_edit": 10 # NFO文件编辑后等待时间（秒），确保文件写入完成
}
```

### 配置参数说明

| 参数名 | 类型 | 说明 | 默认值 |
|-------|------|------|-------|
| `cloud_dir` | 字符串 | 云存储目录路径，处理后的媒体文件会被移动到这里 | `~/Cloud` |
| `tiny_media_manager_dir` | 字符串 | TinyMediaManager的安装目录 | 自动根据操作系统设置 |
| `temp_dir` | 字符串/数组 | 临时目录路径，支持单个或多个目录 | `~/Temp` |
| `tmdb_api_key` | 字符串 | TMDB API密钥，用于获取元数据 | 空（需手动配置） |
| `wait_time_after_scan` | 整数 | 扫描后等待时间（秒），确保所有文件都已准备好 | 30 |
| `wait_time_after_nfo_edit` | 整数 | NFO文件编辑后等待时间（秒），确保文件写入完成 | 10 |

## 使用说明

### 命令行参数

```
Usage:
  -config
        查看或修改配置（当程序目录存在config目录时，会生成基础配置文件）
  -dir string
        指定影片目录路径
  -detect-missing
        检测数据库中所有电视剧的缺失季和剧集
  -nfo string
        指定NFO文件路径
  -scrape-all
        执行所有刮削
  -scrape-movies
        执行电影刮削
  -scrape-tv
        执行电视剧刮削
```

### 使用示例

1. **查看当前配置**：
   ```bash
   ./media-manager -config
   ```

2. **处理单个NFO文件**：
   ```bash
   ./media-manager -nfo /path/to/file.nfo
   ```

3. **处理整个影片目录**：
   ```bash
   ./media-manager -dir /path/to/movies
   ```

4. **执行电影元数据刮削**：
   ```bash
   ./media-manager -scrape-movies
   ```

5. **执行电视剧元数据刮削**：
   ```bash
   ./media-manager -scrape-tv
   ```

6. **执行所有元数据刮削**：
   ```bash
   ./media-manager -scrape-all
   ```

7. **批量检测缺失季和剧集**：
   ```bash
   ./media-manager -detect-missing
   ```

## 编译步骤

### 环境要求
- Go 1.16+ 开发环境
- 支持Cgo的编译器（仅Windows平台需要）

### 使用Makefile编译

项目提供了便捷的Makefile，可以快速编译不同平台的版本：

1. **编译所有平台**：
   ```bash
   make all
   ```

2. **编译Linux版本**：
   ```bash
   make linux
   ```

3. **编译Windows版本**：
   ```bash
   make windows
   ```

4. **编译macOS版本**：
   ```bash
   make macos
   ```

5. **清理编译结果**：
   ```bash
   make clean
   ```

### 手动编译

如果没有Makefile，也可以手动编译：

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o build/media-manager-linux-amd64 .

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags "-s -w" -o build/media-manager-windows-amd64.exe .

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w" -o build/media-manager-darwin-amd64 .
```


### 注意事项

- **配置文件保护**：配置文件中可能包含API密钥等敏感信息，建议不要将 `config/config.json` 文件提交到GitHub
- **编译输出**：编译输出目录 `build/` 不建议提交到GitHub
- **日志文件**：日志目录 `logs/` 不建议提交到GitHub

## 常见问题

### Q: 程序无法启动怎么办？
A: 请检查日志文件（位于 `logs/` 目录）获取详细错误信息，通常是由于配置文件错误或权限问题导致。

### Q: 媒体文件没有被正确移动？
A: 请确保：
1. 配置文件中的 `cloud_dir` 路径正确且有写入权限
2. 临时目录中有有效的NFO文件和媒体文件
3. 媒体文件格式受支持（.mkv, .mp4, .avi, .wmv, .flv, .mov, .rmvb）

### Q: 如何获取TMDB API密钥？
A: 访问 https://www.themoviedb.org/ 注册账号，然后在个人设置中申请API密钥。

## 技术原理

### 🔄 工作流程

1. **参数解析**：解析命令行参数，确定操作类型
2. **单进程检查**：确保程序只有一个实例运行
3. **配置加载**：加载并验证配置文件
4. **操作执行**：根据命令执行相应操作
   - 配置查看：显示当前配置
   - NFO处理：解析和处理NFO文件
   - 刮削操作：从外部源获取元数据
5. **分类移动**：将处理好的媒体文件分类并移动到指定目录

### 🔧 核心模块

- **classifier**：负责媒体文件的分类逻辑，根据NFO文件中的信息确定文件类型和目标目录
- **config**：管理程序配置，支持灵活的配置文件格式和路径
- **parser**：解析NFO文件，提取关键元数据
- **processor**：处理和标准化演员名称和类型信息
- **scraper**：与外部源交互，获取元数据

### 🛡️ 单进程实现

程序通过在配置目录中创建锁文件（`media-manager.lock`）来实现单进程运行。当程序启动时，会尝试创建锁文件，如果失败则表示已经有一个实例在运行。

## 版本信息

当前版本：v1.0.0

## 许可证

MIT License

## 作者

zhangxuan1340

---

**感谢使用Media Manager！** 如果遇到问题或有功能建议，请在GitHub仓库提交Issue。