package migrations

import (
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"gorm.io/gorm"
)

//go:embed *.up.sql
var sqlFS embed.FS

type schemaMigration struct {
	Version string `gorm:"primaryKey;size:128"`
}

func (schemaMigration) TableName() string { return "schema_migrations" }

// Run reads embedded SQL migration files and applies un-applied ones in order.
func Run(db *gorm.DB) error {
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	entries, err := sqlFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, fname := range files {
		version := strings.TrimSuffix(fname, ".up.sql")

		var existing schemaMigration
		if db.Where("version = ?", version).First(&existing).Error == nil {
			slog.Debug("migration already applied", "version", version)
			continue
		}

		content, err := sqlFS.ReadFile(fname)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", fname, err)
		}

		for i, stmt := range splitSQL(string(content)) {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("migration %s stmt %d: %w", fname, i+1, err)
			}
		}

		if err := db.Create(&schemaMigration{Version: version}).Error; err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		slog.Info("applied migration", "version", version)
	}
	return nil
}

func splitSQL(content string) []string {
	var stmts []string
	var buf strings.Builder
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
		if strings.HasSuffix(trimmed, ";") {
			stmts = append(stmts, buf.String())
			buf.Reset()
		}
	}
	if s := strings.TrimSpace(buf.String()); s != "" {
		stmts = append(stmts, s)
	}
	return stmts
}
