package database

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"codebase-indexer/pkg/logger"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migration 迁移结构体
type Migration struct {
	Version     string
	Description string
	SQL         string
}

// Migrator 数据库迁移器
type Migrator struct {
	db         *sql.DB
	logger     logger.Logger
	migrateDir string
}

// NewMigrator 创建新的迁移器
func NewMigrator(db *sql.DB, logger logger.Logger, migrateDir string) *Migrator {
	return &Migrator{
		db:         db,
		logger:     logger,
		migrateDir: migrateDir,
	}
}

// CreateMigrationTable 创建迁移版本表
func (m *Migrator) CreateMigrationTable() error {
	sql := `
		CREATE TABLE IF NOT EXISTS migrations (
			version VARCHAR(255) PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	if _, err := m.db.Exec(sql); err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	m.logger.Info("Migrations table created successfully")
	return nil
}

// GetAppliedMigrations 获取已应用的迁移
func (m *Migrator) GetAppliedMigrations() (map[string]bool, error) {
	rows, err := m.db.Query("SELECT version FROM migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to query applied migrations: %v", err)
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var version string
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("failed to scan migration version: %v", err)
		}
		applied[version] = true
	}

	return applied, nil
}

// GetAvailableMigrations 获取可用的迁移文件
func (m *Migrator) GetAvailableMigrations() ([]Migration, error) {
	// 首先尝试从嵌入的文件系统读取
	files, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		// 如果嵌入文件系统失败，尝试从外部文件系统读取（开发模式）
		if os.DirFS(m.migrateDir) != nil {
			files, err = os.ReadDir(m.migrateDir)
			if err != nil {
				return nil, fmt.Errorf("failed to read migrations directory: %v", err)
			}
			return m.getMigrationsFromOS(files)
		}
		return nil, fmt.Errorf("failed to read embedded migrations: %v", err)
	}

	return m.getMigrationsFromEmbed(files)
}

// getMigrationsFromEmbed 从嵌入的文件系统获取迁移
func (m *Migrator) getMigrationsFromEmbed(files []fs.DirEntry) ([]Migration, error) {
	var migrations []Migration
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 解析文件名格式: 20250610065646_update_tablename_table.sql
		name := file.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// 移除 .sql 后缀
		baseName := strings.TrimSuffix(name, ".sql")

		// 按下划线分割
		parts := strings.Split(baseName, "_")
		if len(parts) < 4 {
			continue
		}

		// 第一部分是时间戳版本号 (14位数字)
		version := parts[0]
		if len(version) != 14 {
			continue
		}

		// 第二部分是操作类型 (create/update/delete)
		action := parts[1]
		if action != "create" && action != "update" && action != "delete" {
			continue
		}

		// 从嵌入的文件系统读取 SQL 内容
		content, err := fs.ReadFile(migrationFS, fmt.Sprintf("migrations/%s", name))
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded migration file %s: %v", name, err)
		}

		migrations = append(migrations, Migration{
			Version:     version,
			Description: baseName,
			SQL:         string(content),
		})
	}

	// 按版本号排序（时间戳）
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// getMigrationsFromOS 从操作系统的文件系统获取迁移（开发模式）
func (m *Migrator) getMigrationsFromOS(files []os.DirEntry) ([]Migration, error) {
	var migrations []Migration
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 解析文件名格式: 20250610065646_update_tablename_table.sql
		name := file.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}

		// 移除 .sql 后缀
		baseName := strings.TrimSuffix(name, ".sql")

		// 按下划线分割
		parts := strings.Split(baseName, "_")
		if len(parts) < 4 {
			continue
		}

		// 第一部分是时间戳版本号 (14位数字)
		version := parts[0]
		if len(version) != 14 {
			continue
		}

		// 第二部分是操作类型 (create/update/delete)
		action := parts[1]
		if action != "create" && action != "update" && action != "delete" {
			continue
		}

		// 从操作系统的文件系统读取 SQL 内容
		content, err := os.ReadFile(filepath.Join(m.migrateDir, name))
		if err != nil {
			return nil, fmt.Errorf("failed to read migration file %s: %v", name, err)
		}

		migrations = append(migrations, Migration{
			Version:     version,
			Description: baseName,
			SQL:         string(content),
		})
	}

	// 按版本号排序（时间戳）
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// ApplyMigration 应用单个迁移
func (m *Migrator) ApplyMigration(migration Migration) error {
	m.logger.Info("Applying migration %s", migration.Description)

	// 开始事务
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 执行迁移 SQL
	if _, err := tx.Exec(migration.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %v", err)
	}

	// 记录迁移版本
	_, err = tx.Exec(
		"INSERT INTO migrations (version, description, applied_at) VALUES (?, ?, ?)",
		migration.Version, migration.Description, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to record migration version: %v", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	m.logger.Info("Migration %s applied successfully", migration.Description)
	return nil
}

// AutoMigrate 自动执行所有未应用的迁移
func (m *Migrator) AutoMigrate() error {
	// 确保迁移表存在
	if err := m.CreateMigrationTable(); err != nil {
		return err
	}

	// 获取已应用的迁移
	applied, err := m.GetAppliedMigrations()
	if err != nil {
		return err
	}

	// 获取可用的迁移
	available, err := m.GetAvailableMigrations()
	if err != nil {
		return err
	}

	// 应用未应用的迁移
	for _, migration := range available {
		if !applied[migration.Version] {
			if err := m.ApplyMigration(migration); err != nil {
				return fmt.Errorf("failed to apply migration %s: %v", migration.Version, err)
			}
		}
	}

	m.logger.Info("Auto migration completed successfully")
	return nil
}

// ExecuteSQLFile 执行外部 SQL 文件
func (m *Migrator) ExecuteSQLFile(filePath string) error {
	m.logger.Info("Executing SQL file: %s", filePath)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("SQL file does not exist: %s", filePath)
	}

	// 读取 SQL 文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file %s: %v", filePath, err)
	}

	// 开始事务
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// 执行 SQL
	if _, err := tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute SQL from file %s: %v", filePath, err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	m.logger.Info("SQL file executed successfully: %s", filePath)
	return nil
}
