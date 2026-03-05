// Package store 管理本地表情库，包括文件系统存储和 SQLite 元数据。
package store

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/world-fish/emoji-transfer/internal/codec"
	_ "modernc.org/sqlite"
)

// Emoji 表示本地表情库中的一条表情记录。
type Emoji struct {
	Hash     string
	Path     string
	Format   codec.Format
	Source   string
	AddedAt  time.Time
}

// Store 管理本地表情库。
type Store struct {
	dir string
	db  *sql.DB
}

// Open 打开（或创建）指定目录下的表情库。
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, "images"), 0755); err != nil {
		return nil, fmt.Errorf("create images dir: %w", err)
	}

	dbPath := filepath.Join(dir, "library.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	s := &Store{dir: dir, db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	return s, nil
}

// Close 关闭表情库。
func (s *Store) Close() error {
	return s.db.Close()
}

// Add 向表情库添加一张表情。若相同 hash 已存在则跳过（自动去重）。
// 返回 true 表示新增，false 表示已存在。
func (s *Store) Add(data []byte, format codec.Format, source string) (bool, *Emoji, error) {
	hash := fmt.Sprintf("%x", md5.Sum(data))

	// 检查是否重复
	var existing string
	err := s.db.QueryRow(`SELECT path FROM emojis WHERE hash = ?`, hash).Scan(&existing)
	if err == nil {
		// 已存在，跳过
		return false, &Emoji{Hash: hash, Path: existing, Format: format, Source: source}, nil
	}
	if err != sql.ErrNoRows {
		return false, nil, fmt.Errorf("query duplicate: %w", err)
	}

	// 写入图片文件
	filename := hash + "." + string(format)
	imgPath := filepath.Join(s.dir, "images", filename)
	if err := os.WriteFile(imgPath, data, 0644); err != nil {
		return false, nil, fmt.Errorf("write image: %w", err)
	}

	// 写入元数据
	now := time.Now()
	_, err = s.db.Exec(
		`INSERT INTO emojis (hash, path, format, source, added_at) VALUES (?, ?, ?, ?, ?)`,
		hash, imgPath, string(format), source, now.Unix(),
	)
	if err != nil {
		return false, nil, fmt.Errorf("insert metadata: %w", err)
	}

	return true, &Emoji{Hash: hash, Path: imgPath, Format: format, Source: source, AddedAt: now}, nil
}

// List 返回所有表情，可按来源平台过滤。
func (s *Store) List(source string) ([]*Emoji, error) {
	query := `SELECT hash, path, format, source, added_at FROM emojis`
	args := []any{}
	if source != "" {
		query += ` WHERE source = ?`
		args = append(args, source)
	}
	query += ` ORDER BY added_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query emojis: %w", err)
	}
	defer rows.Close()

	var emojis []*Emoji
	for rows.Next() {
		e := &Emoji{}
		var addedAtUnix int64
		var format string
		if err := rows.Scan(&e.Hash, &e.Path, &format, &e.Source, &addedAtUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		e.Format = codec.Format(format)
		e.AddedAt = time.Unix(addedAtUnix, 0)
		emojis = append(emojis, e)
	}
	return emojis, rows.Err()
}

// Count 返回表情库中的表情总数。
func (s *Store) Count() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM emojis`).Scan(&n)
	return n, err
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS emojis (
			hash     TEXT PRIMARY KEY,
			path     TEXT NOT NULL,
			format   TEXT NOT NULL,
			source   TEXT NOT NULL,
			added_at INTEGER NOT NULL
		)
	`)
	return err
}
