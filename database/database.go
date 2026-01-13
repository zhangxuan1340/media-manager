package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/user/media-manager/utils"
)

// MediaRecord 表示媒体记录的结构
type MediaRecord struct {
	ID            int       `db:"id"`
	FileName      string    `db:"file_name"`
	Title         string    `db:"title"`
	OriginalTitle string    `db:"original_title"`
	Year          string    `db:"year"`
	Country       string    `db:"country"`
	Genres        string    `db:"genres"`
	Actors        string    `db:"actors"`
	Category      string    `db:"category"`
	SourcePath    string    `db:"source_path"`
	TargetPath    string    `db:"target_path"`
	ProcessedAt   time.Time `db:"processed_at"`
	Runtime       string    `db:"runtime"`
	Plot          string    `db:"plot"`
	IMDbID        string    `db:"imdb_id"`
	TMDbID        string    `db:"tmdb_id"`
	Season        string    `db:"season"`
	Episode       string    `db:"episode"`
	Director      string    `db:"director"`
	Writer        string    `db:"writer"`
	Rating        string    `db:"rating"`
}

// DB 是数据库连接的全局变量
var DB *sql.DB

// GetDatabasePath 获取数据库文件路径
func GetDatabasePath() string {
	var dataDir string
	var err error

	// 1. 首先检查用户当前目录下是否存在Data目录（只检查不创建）
	currentDir, err := os.Getwd()
	if err == nil {
		dataDir = filepath.Join(currentDir, "Data")
		if _, err := os.Stat(dataDir); err == nil {
			return filepath.Join(dataDir, "media_manager.db")
		}
	}

	// 2. 检查程序执行文件所在目录下是否存在Data目录（只检查不创建）
	exeDir, err := utils.GetExecutableDir()
	if err == nil {
		dataDir = filepath.Join(exeDir, "Data")
		if _, err := os.Stat(dataDir); err == nil {
			return filepath.Join(dataDir, "media_manager.db")
		}
	}

	// 3. 最后使用用户主目录下的.media-manager/Data目录（不存在则创建）
	homeDir, err := os.UserHomeDir()
	if err == nil {
		dataDir = filepath.Join(homeDir, ".media-manager", "Data")
		// 确保用户主目录下的Data目录存在
		if err := os.MkdirAll(dataDir, 0755); err == nil {
			return filepath.Join(dataDir, "media_manager.db")
		}
	}

	// 如果所有尝试都失败，输出错误并退出
	fmt.Printf("无法创建Data目录\n")
	os.Exit(1)
	return "" // 永远不会执行到这里
}

// InitDatabase 初始化数据库
func InitDatabase() {
	dbPath := GetDatabasePath()

	// 打开数据库连接
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("无法打开数据库: %v\n", err)
		os.Exit(1)
	}

	// 验证数据库连接
	if err := db.Ping(); err != nil {
		fmt.Printf("无法连接到数据库: %v\n", err)
		os.Exit(1)
	}

	// 创建媒体记录表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS media_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_name TEXT,
		title TEXT,
		original_title TEXT,
		year TEXT,
		country TEXT,
		genres TEXT,
		actors TEXT,
		category TEXT,
		source_path TEXT,
		target_path TEXT,
		processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		runtime TEXT,
		plot TEXT,
		imdb_id TEXT,
		tmdb_id TEXT,
		season TEXT,
		episode TEXT,
		director TEXT,
		writer TEXT,
		rating TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		fmt.Printf("无法创建媒体记录表: %v\n", err)
		os.Exit(1)
	}

	// 单独检查并添加year字段（针对旧表）
	// 查询表结构，检查year字段是否存在
	rows, err := db.Query(`PRAGMA table_info(media_records)`)
	if err != nil {
		fmt.Printf("查询表结构失败: %v\n", err)
	} else {
		defer rows.Close()

		var hasYearField bool
		var cid int
		var name string
		var dataType string
		var notNull int
		var dfltValue interface{}
		var pk int

		for rows.Next() {
			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err == nil {
				if name == "year" {
					hasYearField = true
					break
				}
			}
		}

		// 如果year字段不存在，尝试添加
		if !hasYearField {
			_, err = db.Exec(`ALTER TABLE media_records ADD COLUMN year TEXT;`)
			if err != nil {
				// 忽略字段已存在的错误
				if !strings.Contains(err.Error(), "duplicate column name") {
					fmt.Printf("添加year字段失败: %v\n", err)
				}
			}
		}
	}

	// 为已存在的表添加其他新字段
	// 使用ALTER TABLE语句添加字段，如果字段已存在会被忽略
	alterTableStatements := []string{
		`ALTER TABLE media_records ADD COLUMN runtime TEXT;`,
		`ALTER TABLE media_records ADD COLUMN plot TEXT;`,
		`ALTER TABLE media_records ADD COLUMN imdb_id TEXT;`,
		`ALTER TABLE media_records ADD COLUMN tmdb_id TEXT;`,
		`ALTER TABLE media_records ADD COLUMN season TEXT;`,
		`ALTER TABLE media_records ADD COLUMN episode TEXT;`,
		`ALTER TABLE media_records ADD COLUMN director TEXT;`,
		`ALTER TABLE media_records ADD COLUMN writer TEXT;`,
		`ALTER TABLE media_records ADD COLUMN rating TEXT;`,
	}

	for _, stmt := range alterTableStatements {
		_, err = db.Exec(stmt)
		if err != nil {
			// 忽略字段已存在的错误
			if strings.Contains(err.Error(), "duplicate column name") {
				continue
			}
			// 其他错误需要输出
			fmt.Printf("执行ALTER TABLE语句失败: %v\n", err)
			// 不退出，继续执行其他语句
		}
	}

	// 设置全局数据库连接
	DB = db
}

// InsertMediaRecord 插入媒体记录
func InsertMediaRecord(record *MediaRecord) error {
	if DB == nil {
		InitDatabase()
	}

	insertSQL := `
	INSERT INTO media_records (file_name, title, original_title, year, country, genres, actors, category, source_path, target_path, processed_at, runtime, plot, imdb_id, tmdb_id, season, episode, director, writer, rating) 
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := DB.Exec(insertSQL,
		record.FileName,
		record.Title,
		record.OriginalTitle,
		record.Year,
		record.Country,
		record.Genres,
		record.Actors,
		record.Category,
		record.SourcePath,
		record.TargetPath,
		record.ProcessedAt,
		record.Runtime,
		record.Plot,
		record.IMDbID,
		record.TMDbID,
		record.Season,
		record.Episode,
		record.Director,
		record.Writer,
		record.Rating,
	)

	return err
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() {
	if DB != nil {
		DB.Close()
	}
}
