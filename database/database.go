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
	UpdatedAt     time.Time `db:"updated_at"`
	Runtime       string    `db:"runtime"`
	Plot          string    `db:"plot"`
	IMDbID        string    `db:"imdb_id"`
	TMDbID        string    `db:"tmdb_id"`
	Season        string    `db:"season"`
	Episode       string    `db:"episode"`
	Director      string    `db:"director"`
	Writer        string    `db:"writer"`
	Rating        string    `db:"rating"`
	Resolution    string    `db:"resolution"`
	Version       int       `db:"version"`
	IsComplete    bool      `db:"is_complete"`
}

// MissingEpisode 表示缺失的剧集记录
type MissingEpisode struct {
	ID            int       `db:"id"`
	MediaID       int       `db:"media_id"`
	Title         string    `db:"title"`
	OriginalTitle string    `db:"original_title"`
	TMDbID        string    `db:"tmdb_id"`
	Season        int       `db:"season"`
	Episode       int       `db:"episode"`
	DetectedAt    time.Time `db:"detected_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	Status        string    `db:"status"`
}

// MissingSeason 表示缺失的季记录
type MissingSeason struct {
	ID            int       `db:"id"`
	MediaID       int       `db:"media_id"`
	Title         string    `db:"title"`
	OriginalTitle string    `db:"original_title"`
	TMDbID        string    `db:"tmdb_id"`
	Season        int       `db:"season"`
	DetectedAt    time.Time `db:"detected_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	Status        string    `db:"status"`
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
	// 检查DB是否已经初始化，这是关键的幂等性检查
	if DB != nil {
		return
	}

	dbPath := GetDatabasePath()

	// 打开数据库连接
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("无法打开数据库: %v\n", err)
		os.Exit(1)
	}

	// 设置SQLite连接参数
	// 使用简单的连接池配置，避免死锁
	db.SetMaxOpenConns(1)    // 只允许一个连接，避免SQLite锁定问题
	db.SetMaxIdleConns(0)    // 不保留空闲连接
	db.SetConnMaxLifetime(0) // 连接永不超时

	// 立即设置全局DB变量，避免并发初始化
	DB = db

	// 验证数据库连接
	if err := db.Ping(); err != nil {
		fmt.Printf("无法连接到数据库: %v\n", err)
		DB = nil // 重置DB，以便下次可以重试
		os.Exit(1)
	}

	// 创建媒体记录表，包含所有必要字段
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
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		runtime TEXT,
		plot TEXT,
		imdb_id TEXT,
		tmdb_id TEXT,
		season TEXT,
		episode TEXT,
		director TEXT,
		writer TEXT,
		rating TEXT,
		resolution TEXT,
		version INTEGER DEFAULT 1,
		is_complete BOOLEAN DEFAULT FALSE
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		fmt.Printf("无法创建媒体记录表: %v\n", err)
		// 不退出，继续执行
	}
	// 无论表是否创建成功，都检查并添加缺少的字段
	// 这确保了旧表也会被更新为包含所有必要字段

	// 如果表已经存在，检查并添加缺少的字段
	// 这里我们使用更安全的方式，避免锁定问题
	// 只检查和添加必要的字段，使用简单的ALTER TABLE语句
	addMissingField := func(fieldName, fieldType string) {
		// 使用PRAGMA table_info检查字段是否存在
		var exists bool
		rows, err := db.Query(`PRAGMA table_info(media_records)`)
		if err != nil {
			fmt.Printf("查询表结构失败: %v\n", err)
			return
		}

		for rows.Next() {
			var cid int
			var name string
			var dataType string
			var notNull int
			var dfltValue interface{}
			var pk int
			if err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				fmt.Printf("扫描表结构失败: %v\n", err)
				break
			}
			if name == fieldName {
				exists = true
				break
			}
		}
		rows.Close()

		// 如果字段不存在，尝试添加
		if !exists {
			// 使用简单的ALTER TABLE语句，不使用默认值
			alterSQL := fmt.Sprintf("ALTER TABLE media_records ADD COLUMN %s %s;", fieldName, fieldType)
			if _, err := db.Exec(alterSQL); err != nil {
				// 忽略添加字段的错误，特别是"duplicate column name"错误
				if !strings.Contains(err.Error(), "duplicate column name") {
					fmt.Printf("添加字段 %s 失败: %v\n", fieldName, err)
				}
			}
		}
	}

	// 添加可能缺少的字段
	addMissingField("updated_at", "TIMESTAMP")
	addMissingField("resolution", "TEXT")
	addMissingField("version", "INTEGER")
	addMissingField("is_complete", "BOOLEAN")

	// 创建缺失剧集表
	createMissingEpisodesTableSQL := `
	CREATE TABLE IF NOT EXISTS missing_episodes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_id INTEGER,
		title TEXT,
		original_title TEXT,
		tmdb_id TEXT,
		season INTEGER,
		episode INTEGER,
		detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'missing',
		FOREIGN KEY (media_id) REFERENCES media_records (id)
	);`

	if _, err := db.Exec(createMissingEpisodesTableSQL); err != nil {
		fmt.Printf("无法创建缺失剧集表: %v\n", err)
		// 不退出，继续执行
	}

	// 创建缺失季表
	createMissingSeasonsTableSQL := `
	CREATE TABLE IF NOT EXISTS missing_seasons (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		media_id INTEGER,
		title TEXT,
		original_title TEXT,
		tmdb_id TEXT,
		season INTEGER,
		detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'missing',
		FOREIGN KEY (media_id) REFERENCES media_records (id)
	);`

	if _, err := db.Exec(createMissingSeasonsTableSQL); err != nil {
		fmt.Printf("无法创建缺失季表: %v\n", err)
		// 不退出，继续执行
	}
}

// InsertOrUpdateMediaRecord 插入或更新媒体记录
func InsertOrUpdateMediaRecord(record *MediaRecord) error {
	if DB == nil {
		InitDatabase()
	}

	// 检查是否已存在相同的媒体记录
	var existingID int
	var existingVersion int

	// 对于电影，使用标题、年份和分辨率作为唯一标识
	// 对于电视剧，使用标题、年份、季数和分辨率作为唯一标识
	var query string
	var args []interface{}

	if record.Season == "" {
		// 电影
		query = `SELECT id, version FROM media_records WHERE title = ? AND year = ? AND category NOT LIKE '%Show'`
		args = []interface{}{record.Title, record.Year}
	} else {
		// 电视剧
		query = `SELECT id, version FROM media_records WHERE title = ? AND year = ? AND season = ? AND category LIKE '%Show'`
		args = []interface{}{record.Title, record.Year, record.Season}
	}

	err := DB.QueryRow(query, args...).Scan(&existingID, &existingVersion)

	if err != nil {
		if err == sql.ErrNoRows {
			// 记录不存在，执行插入
			// 设置默认值
			now := time.Now()
			version := 1
			isComplete := false
			if record.Version > 0 {
				version = record.Version
			}
			if record.IsComplete {
				isComplete = record.IsComplete
			}

			insertSQL := `
			INSERT INTO media_records (file_name, title, original_title, year, country, genres, actors, category, source_path, target_path, processed_at, updated_at, runtime, plot, imdb_id, tmdb_id, season, episode, director, writer, rating, resolution, version, is_complete) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

			_, err = DB.Exec(insertSQL,
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
				now,
				now,
				record.Runtime,
				record.Plot,
				record.IMDbID,
				record.TMDbID,
				record.Season,
				record.Episode,
				record.Director,
				record.Writer,
				record.Rating,
				record.Resolution,
				version,
				isComplete,
			)

			return err
		} else {
			return err
		}
	} else {
		// 记录存在，执行更新
		updateSQL := `
		UPDATE media_records SET 
			file_name = ?, 
			original_title = ?, 
			country = ?, 
			genres = ?, 
			actors = ?, 
			category = ?, 
			source_path = ?, 
			target_path = ?, 
			updated_at = ?, 
			runtime = ?, 
			plot = ?, 
			imdb_id = ?, 
			tmdb_id = ?, 
			episode = ?, 
			director = ?, 
			writer = ?, 
			rating = ?, 
			resolution = ?, 
			version = ?, 
			is_complete = ? 
		WHERE id = ?`

		_, err = DB.Exec(updateSQL,
			record.FileName,
			record.OriginalTitle,
			record.Country,
			record.Genres,
			record.Actors,
			record.Category,
			record.SourcePath,
			record.TargetPath,
			time.Now(),
			record.Runtime,
			record.Plot,
			record.IMDbID,
			record.TMDbID,
			record.Episode,
			record.Director,
			record.Writer,
			record.Rating,
			record.Resolution,
			existingVersion+1, // 版本号递增
			record.IsComplete,
			existingID,
		)

		return err
	}
}

// InsertMissingSeason 插入缺失季记录
func InsertMissingSeason(record *MissingSeason) error {
	if DB == nil {
		InitDatabase()
	}

	// 检查是否已存在相同的缺失季记录
	var existingID int
	query := `SELECT id FROM missing_seasons WHERE title = ? AND tmdb_id = ? AND season = ? AND status = 'missing'`
	err := DB.QueryRow(query, record.Title, record.TMDbID, record.Season).Scan(&existingID)

	if err != nil {
		if err == sql.ErrNoRows {
			// 记录不存在，执行插入
			insertSQL := `
			INSERT INTO missing_seasons (media_id, title, original_title, tmdb_id, season, detected_at, updated_at, status) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

			_, err = DB.Exec(insertSQL,
				record.MediaID,
				record.Title,
				record.OriginalTitle,
				record.TMDbID,
				record.Season,
				time.Now(),
				time.Now(),
				"missing",
			)

			return err
		} else {
			return err
		}
	}

	return nil
}

// InsertMissingEpisode 插入缺失剧集记录
func InsertMissingEpisode(record *MissingEpisode) error {
	if DB == nil {
		InitDatabase()
	}

	// 检查是否已存在相同的缺失剧集记录
	var existingID int
	query := `SELECT id FROM missing_episodes WHERE title = ? AND tmdb_id = ? AND season = ? AND episode = ? AND status = 'missing'`
	err := DB.QueryRow(query, record.Title, record.TMDbID, record.Season, record.Episode).Scan(&existingID)

	if err != nil {
		if err == sql.ErrNoRows {
			// 记录不存在，执行插入
			insertSQL := `
			INSERT INTO missing_episodes (media_id, title, original_title, tmdb_id, season, episode, detected_at, updated_at, status) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

			_, err = DB.Exec(insertSQL,
				record.MediaID,
				record.Title,
				record.OriginalTitle,
				record.TMDbID,
				record.Season,
				record.Episode,
				time.Now(),
				time.Now(),
				"missing",
			)

			return err
		} else {
			return err
		}
	}

	return nil
}

// UpdateMissingItemStatus 更新缺失项目的状态
func UpdateMissingItemStatus(table string, id int, status string) error {
	if DB == nil {
		InitDatabase()
	}

	updateSQL := `
	UPDATE ` + table + ` SET 
		status = ?, 
		updated_at = ? 
	WHERE id = ?`

	_, err := DB.Exec(updateSQL, status, time.Now(), id)
	return err
}

// GetMissingSeasons 获取所有缺失的季记录
func GetMissingSeasons(filter map[string]interface{}) ([]MissingSeason, error) {
	if DB == nil {
		InitDatabase()
	}

	var missingSeasons []MissingSeason
	query := `SELECT id, media_id, title, original_title, tmdb_id, season, detected_at, updated_at, status FROM missing_seasons WHERE status = 'missing'`

	// 添加过滤条件
	var args []interface{}
	if title, ok := filter["title"].(string); ok && title != "" {
		query += ` AND title LIKE ?`
		args = append(args, "%"+title+"%")
	}

	if tmdbID, ok := filter["tmdb_id"].(string); ok && tmdbID != "" {
		query += ` AND tmdb_id = ?`
		args = append(args, tmdbID)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var season MissingSeason
		if err := rows.Scan(&season.ID, &season.MediaID, &season.Title, &season.OriginalTitle, &season.TMDbID, &season.Season, &season.DetectedAt, &season.UpdatedAt, &season.Status); err != nil {
			return nil, err
		}
		missingSeasons = append(missingSeasons, season)
	}

	return missingSeasons, nil
}

// GetMissingEpisodes 获取所有缺失的剧集记录
func GetMissingEpisodes(filter map[string]interface{}) ([]MissingEpisode, error) {
	if DB == nil {
		InitDatabase()
	}

	var missingEpisodes []MissingEpisode
	query := `SELECT id, media_id, title, original_title, tmdb_id, season, episode, detected_at, updated_at, status FROM missing_episodes WHERE status = 'missing'`

	// 添加过滤条件
	var args []interface{}
	if title, ok := filter["title"].(string); ok && title != "" {
		query += ` AND title LIKE ?`
		args = append(args, "%"+title+"%")
	}

	if tmdbID, ok := filter["tmdb_id"].(string); ok && tmdbID != "" {
		query += ` AND tmdb_id = ?`
		args = append(args, tmdbID)
	}

	if season, ok := filter["season"].(int); ok && season > 0 {
		query += ` AND season = ?`
		args = append(args, season)
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var episode MissingEpisode
		if err := rows.Scan(&episode.ID, &episode.MediaID, &episode.Title, &episode.OriginalTitle, &episode.TMDbID, &episode.Season, &episode.Episode, &episode.DetectedAt, &episode.UpdatedAt, &episode.Status); err != nil {
			return nil, err
		}
		missingEpisodes = append(missingEpisodes, episode)
	}

	return missingEpisodes, nil
}

// GetMediaRecords 获取媒体记录列表
func GetMediaRecords(filter map[string]interface{}) ([]MediaRecord, error) {
	if DB == nil {
		InitDatabase()
	}

	var mediaRecords []MediaRecord
	// 使用简单的SELECT语句，不使用COALESCE，避免类型转换问题
	query := `SELECT 
		id, 
		file_name, 
		title, 
		original_title, 
		year, 
		country, 
		genres, 
		actors, 
		category, 
		source_path, 
		target_path, 
		processed_at, 
		updated_at, 
		runtime, 
		plot, 
		imdb_id, 
		tmdb_id, 
		season, 
		episode, 
		director, 
		writer, 
		rating, 
		resolution, 
		version, 
		is_complete 
	FROM media_records`

	// 添加过滤条件
	var args []interface{}
	if title, ok := filter["title"].(string); ok && title != "" {
		query += ` WHERE title LIKE ?`
		args = append(args, "%"+title+"%")
	}

	if isComplete, ok := filter["is_complete"].(bool); ok {
		if len(args) > 0 {
			query += ` AND is_complete = ?`
		} else {
			query += ` WHERE is_complete = ?`
		}
		args = append(args, isComplete)
	}

	if category, ok := filter["category"].(string); ok && category != "" {
		if len(args) > 0 {
			query += ` AND category LIKE ?`
		} else {
			query += ` WHERE category LIKE ?`
		}
		args = append(args, "%"+category+"%")
	}

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 定义临时结构体，用于处理可能为NULL的字段
	type tempMediaRecord struct {
		ID            int
		FileName      *string
		Title         *string
		OriginalTitle *string
		Year          *string
		Country       *string
		Genres        *string
		Actors        *string
		Category      *string
		SourcePath    *string
		TargetPath    *string
		ProcessedAt   time.Time
		UpdatedAt     *time.Time
		Runtime       *string
		Plot          *string
		IMDbID        *string
		TMDbID        *string
		Season        *string
		Episode       *string
		Director      *string
		Writer        *string
		Rating        *string
		Resolution    *string
		Version       *int
		IsComplete    *bool
	}

	for rows.Next() {
		var temp tempMediaRecord
		// 使用指针类型扫描，允许NULL值
		if err := rows.Scan(
			&temp.ID,
			&temp.FileName,
			&temp.Title,
			&temp.OriginalTitle,
			&temp.Year,
			&temp.Country,
			&temp.Genres,
			&temp.Actors,
			&temp.Category,
			&temp.SourcePath,
			&temp.TargetPath,
			&temp.ProcessedAt,
			&temp.UpdatedAt,
			&temp.Runtime,
			&temp.Plot,
			&temp.IMDbID,
			&temp.TMDbID,
			&temp.Season,
			&temp.Episode,
			&temp.Director,
			&temp.Writer,
			&temp.Rating,
			&temp.Resolution,
			&temp.Version,
			&temp.IsComplete,
		); err != nil {
			return nil, err
		}

		// 将临时结构体转换为MediaRecord，处理NULL值
		record := MediaRecord{
			ID:          temp.ID,
			ProcessedAt: temp.ProcessedAt,
		}

		// 处理可能为NULL的字符串字段
		if temp.FileName != nil {
			record.FileName = *temp.FileName
		}
		if temp.Title != nil {
			record.Title = *temp.Title
		}
		if temp.OriginalTitle != nil {
			record.OriginalTitle = *temp.OriginalTitle
		}
		if temp.Year != nil {
			record.Year = *temp.Year
		}
		if temp.Country != nil {
			record.Country = *temp.Country
		}
		if temp.Genres != nil {
			record.Genres = *temp.Genres
		}
		if temp.Actors != nil {
			record.Actors = *temp.Actors
		}
		if temp.Category != nil {
			record.Category = *temp.Category
		}
		if temp.SourcePath != nil {
			record.SourcePath = *temp.SourcePath
		}
		if temp.TargetPath != nil {
			record.TargetPath = *temp.TargetPath
		}
		if temp.UpdatedAt != nil {
			record.UpdatedAt = *temp.UpdatedAt
		} else {
			// 如果updated_at为NULL，使用当前时间
			record.UpdatedAt = time.Now()
		}
		if temp.Runtime != nil {
			record.Runtime = *temp.Runtime
		}
		if temp.Plot != nil {
			record.Plot = *temp.Plot
		}
		if temp.IMDbID != nil {
			record.IMDbID = *temp.IMDbID
		}
		if temp.TMDbID != nil {
			record.TMDbID = *temp.TMDbID
		}
		if temp.Season != nil {
			record.Season = *temp.Season
		}
		if temp.Episode != nil {
			record.Episode = *temp.Episode
		}
		if temp.Director != nil {
			record.Director = *temp.Director
		}
		if temp.Writer != nil {
			record.Writer = *temp.Writer
		}
		if temp.Rating != nil {
			record.Rating = *temp.Rating
		}
		if temp.Resolution != nil {
			record.Resolution = *temp.Resolution
		}
		if temp.Version != nil {
			record.Version = *temp.Version
		} else {
			// 如果version为NULL，使用默认值1
			record.Version = 1
		}
		if temp.IsComplete != nil {
			record.IsComplete = *temp.IsComplete
		} else {
			// 如果is_complete为NULL，使用默认值false
			record.IsComplete = false
		}

		mediaRecords = append(mediaRecords, record)
	}

	return mediaRecords, nil
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() {
	if DB != nil {
		DB.Close()
	}
}
