# 媒体管理程序 (Media Manager)

一个功能强大的媒体管理工具，用于自动化管理和组织电影、电视剧等媒体文件。

## 功能特性

- **媒体刮削**：集成tinyMediaManager执行电影和电视剧信息刮削
- **NFO文件处理**：自动检查和翻译媒体类型，验证演员名称
- **智能分类**：根据国家/地区、类型自动将媒体文件分类到对应目录
- **多平台支持**：支持Windows、macOS和Linux系统
- **多目录管理**：支持多个临时目录配置
- **日志记录**：详细的操作日志，便于问题排查
- **数据库记录**：记录媒体文件信息到数据库

## 目录结构

```
media-manager/
├── build/                 # 编译输出目录
├── classifier/            # 分类和移动模块
│   └── classifier.go      # 媒体分类和移动逻辑
├── config/                # 配置模块
│   ├── config.go          # 配置加载和保存
│   ├── config.json        # 配置文件
│   └── media-manager.lock # 单进程锁文件
├── database/              # 数据库模块
│   └── database.go        # 媒体信息数据库操作
├── logging/               # 日志模块
│   └── logging.go         # 日志记录功能
├── logs/                  # 日志文件目录
├── parser/                # NFO文件解析模块
│   └── nfo.go             # NFO文件解析逻辑
├── processor/             # 数据处理模块
│   ├── actor.go           # 演员信息处理
│   └── genre.go           # 类型信息处理
├── scraper/               # 刮削模块
│   └── scraper.go         # 媒体信息刮削逻辑
├── tmdb/                  # TMDB API模块
│   └── tmdb.go            # TMDB API调用
├── utils/                 # 工具函数模块
│   ├── chinese.go         # 中文处理工具
│   └── path.go            # 路径处理工具
├── Makefile               # 编译脚本
├── go.mod                 # Go模块定义
├── go.sum                 # Go依赖锁定
├── main.go                # 程序入口
├── singleprocess_unix.go  # Unix平台单进程实现
└── singleprocess_windows.go # Windows平台单进程实现
```

## 配置说明

配置文件位于以下位置之一（按优先级顺序）：
1. 当前目录下的 `config/config.json`
2. 程序执行文件所在目录下的 `config/config.json`
3. 用户主目录下的 `.media-manager/config.json`

### 配置项说明

```json
{
  "cloud_dir": "~/Cloud",             # 媒体文件最终存放目录
  "tiny_media_manager_dir": "/usr/local/bin", # tinyMediaManager安装目录
  "temp_dir": "~/Temp",              # 临时目录（支持单个字符串或数组）
  "tmdb_api_key": "",                # TMDB API密钥（可选）
  "wait_time_after_scan": 30,         # 刮削完成后等待时间（秒）
  "wait_time_after_nfo_edit": 10      # NFO文件编辑完成后等待时间（秒）
}
```

#### 详细配置项解释

- **cloud_dir**：媒体文件分类后最终移动到的云盘目录
- **tiny_media_manager_dir**：tinyMediaManager工具的安装目录
  - Windows: `"C:\\Program Files\\tinyMediaManager"`
  - macOS: `"/Applications/tinyMediaManager.app/Contents/MacOS"`
  - Linux: `"/usr/local/bin"`
- **temp_dir**：临时目录，支持单个目录或多个目录数组
  - 示例：`"~/Temp"` 或 `["~/Temp1", "~/Temp2"]`
- **tmdb_api_key**：可选的TMDB API密钥，用于获取更准确的制作国家信息
- **wait_time_after_scan**：刮削完成后等待的秒数，确保所有刮削数据已写入磁盘
- **wait_time_after_nfo_edit**：NFO文件编辑完成后等待的秒数，确保文件已保存

## 安装和编译

### 前置依赖

- Go 1.16+（用于编译）
- tinyMediaManager（用于媒体信息刮削）

### 编译步骤

1. 克隆项目到本地

```bash
git clone https://github.com/zhangxuan1340/media-manager.git
cd media-manager
```

2. 使用Makefile编译

```bash
# 编译所有平台版本
make all

# 编译Linux版本
make linux

# 编译Windows版本
make windows

# 编译macOS版本
make macos

# 清理编译结果
make clean
```

3. 编译结果将输出到 `build/` 目录

## 使用方法

### 命令行参数

```bash
./media-manager [选项]
```

#### 主要选项

- **-nfo**：指定单个NFO文件路径
- **-dir**：指定影片目录路径
- **-scrape-movies**：执行电影刮削
- **-scrape-tv**：执行电视剧刮削
- **-scrape-all**：执行所有刮削
- **-config**：查看当前配置

### 使用示例

1. **查看配置**

```bash
./media-manager -config
```

2. **执行刮削**

```bash
# 刮削电影
./media-manager -scrape-movies

# 刮削电视剧
./media-manager -scrape-tv

# 刮削所有媒体
./media-manager -scrape-all
```

3. **处理单个NFO文件**

```bash
./media-manager -nfo /path/to/movie.nfo
```

4. **处理目录中的所有NFO文件**

```bash
./media-manager -dir /path/to/movies
```

## 工作原理

### 媒体刮削流程

1. 用户执行刮削命令（如 `-scrape-movies`）
2. 程序调用tinyMediaManager工具在临时目录执行刮削
3. 刮削完成后等待指定时间（`wait_time_after_scan`）
4. 扫描临时目录中的NFO文件
5. 处理每个NFO文件：
   - 检查和翻译媒体类型
   - 验证演员名称
   - 自动分类并移动媒体文件到最终目录

### 分类逻辑

媒体文件根据以下规则进行分类：

1. **纪录片**：包含"纪录片"或"documentary"类型
2. **综艺节目**：包含综艺相关关键词（如"综艺节目"、"真人秀"等）
3. **动漫**：包含"动漫"、"动画"或"animation"类型
4. **日本/韩国**：制作国家为日本或韩国
5. **国内**：制作国家为中国（中国大陆、香港、台湾）
6. **其他国家**：其他国家/地区的媒体

分类后将媒体文件移动到对应子目录：
- `CnMovie`：国内电影
- `CnShow`：国内电视剧
- `EnMovie`：其他国家电影
- `EnShow`：其他国家电视剧
- `Jp&KrMovie`：日本/韩国电影
- `Jp&KrShow`：日本/韩国电视剧
- `DmMovie`：动漫电影
- `DmShow`：动漫剧集
- `JlShow`：纪录片
- `XSShow`：综艺节目

### NFO文件处理

1. **类型处理**：检查类型是否为简体中文，非中文类型将自动翻译
2. **演员处理**：检查演员名称是否为中文，非中文演员将记录到报告文件
3. **多NFO处理**：检查目录中是否有多个NFO文件，避免重复处理

## 技术栈

- **语言**：Go 1.16+
- **编译工具**：Makefile
- **依赖管理**：Go Modules
- **媒体刮削**：tinyMediaManager
- **API集成**：TMDB API（可选）

## 注意事项

1. 确保tinyMediaManager已正确安装并配置
2. 配置文件中的路径使用绝对路径或使用 `~` 表示用户主目录
3. 程序执行时需要对临时目录和目标目录有读写权限
4. 多目录配置时，刮削仅在第一个临时目录执行
5. 程序使用单进程锁确保同一时间只有一个实例运行

## 日志

日志文件存储在 `logs/` 目录下，按日期命名（如 `2026-01-09.log`）。日志包含程序启动、配置加载、刮削执行、NFO文件处理等详细信息。

## 许可证

MIT License