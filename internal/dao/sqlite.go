package dao

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	SQL *sql.DB
}

type FileMeta struct {
	ID        int64
	SourceURL string
	Filename  string
	Size      int64
	MimeType  string
	ETag      string
	LastMod   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func OpenSQLite(path string) (*DB, error) {
	dsn := path
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &DB{SQL: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS files (
	  id INTEGER PRIMARY KEY AUTOINCREMENT,
	  source_url TEXT NOT NULL UNIQUE,
	  filename TEXT,
	  size INTEGER,
	  mime_type TEXT,
	  etag TEXT,
	  last_mod TEXT,
	  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TRIGGER IF NOT EXISTS files_updated_at
	AFTER UPDATE ON files
	BEGIN
	  UPDATE files SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
	END;

    -- 后台用户表
    CREATE TABLE IF NOT EXISTS users (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      username TEXT NOT NULL UNIQUE,
      password_hash TEXT NOT NULL,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );

    -- 生成的链接（签名）记录
    CREATE TABLE IF NOT EXISTS links (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      source_url TEXT NOT NULL,
      signed_path TEXT NOT NULL,
      fingerprint TEXT,
      expire_at TIMESTAMP NOT NULL,
      created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
    CREATE INDEX IF NOT EXISTS idx_links_signed_path ON links(signed_path);
    CREATE INDEX IF NOT EXISTS idx_links_fingerprint ON links(fingerprint);

    -- 下载/访问事件
    CREATE TABLE IF NOT EXISTS link_events (
      id INTEGER PRIMARY KEY AUTOINCREMENT,
      link_id INTEGER NOT NULL,
      at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
      ip TEXT,
      ua TEXT,
      fingerprint TEXT,
      bytes INTEGER,
      suspicious INTEGER DEFAULT 0,
      FOREIGN KEY(link_id) REFERENCES links(id)
    );
	`)
	return err
}

func (d *DB) UpsertFileMeta(ctx context.Context, m *FileMeta) error {
	if m == nil {
		return errors.New("nil meta")
	}
	_, err := d.SQL.ExecContext(ctx, `
	INSERT INTO files (source_url, filename, size, mime_type, etag, last_mod)
	VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(source_url) DO UPDATE SET
	  filename=excluded.filename,
	  size=excluded.size,
	  mime_type=excluded.mime_type,
	  etag=excluded.etag,
	  last_mod=excluded.last_mod
	`, m.SourceURL, m.Filename, m.Size, m.MimeType, m.ETag, m.LastMod)
	return err
}

func (d *DB) GetBySourceURL(ctx context.Context, source string) (*FileMeta, error) {
	row := d.SQL.QueryRowContext(ctx, `
	SELECT id, source_url, filename, size, mime_type, etag, last_mod, created_at, updated_at
	FROM files WHERE source_url = ?
	`, source)
	m := &FileMeta{}
	if err := row.Scan(&m.ID, &m.SourceURL, &m.Filename, &m.Size, &m.MimeType, &m.ETag, &m.LastMod, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return nil, err
	}
	return m, nil
}

// 用户
func (d *DB) EnsureAdmin(ctx context.Context, username, passwordHash string) error {
	// 若不存在则创建
	_, err := d.SQL.ExecContext(ctx, `INSERT OR IGNORE INTO users (username, password_hash) VALUES (?, ?)`, username, passwordHash)
	return err
}

func (d *DB) GetUserByName(ctx context.Context, username string) (id int64, passhash string, err error) {
	row := d.SQL.QueryRowContext(ctx, `SELECT id, password_hash FROM users WHERE username=?`, username)
	err = row.Scan(&id, &passhash)
	return
}

// 链接与事件
func (d *DB) InsertLink(ctx context.Context, sourceURL, signedPath, fingerprint string, expireAt time.Time) error {
	_, err := d.SQL.ExecContext(ctx, `INSERT INTO links (source_url, signed_path, fingerprint, expire_at) VALUES (?, ?, ?, ?)`, sourceURL, signedPath, fingerprint, expireAt)
	return err
}

func (d *DB) GetLinkBySignedPath(ctx context.Context, signedPath string) (int64, string, string, time.Time, error) {
	row := d.SQL.QueryRowContext(ctx, `SELECT id, source_url, fingerprint, expire_at FROM links WHERE signed_path=? ORDER BY id DESC LIMIT 1`, signedPath)
	var id int64
	var src, fp string
	var exp time.Time
	err := row.Scan(&id, &src, &fp, &exp)
	return id, src, fp, exp, err
}

func (d *DB) InsertLinkEvent(ctx context.Context, linkID int64, ip, ua, fingerprint string, bytes int64, suspicious bool) error {
	sus := 0
	if suspicious {
		sus = 1
	}
	_, err := d.SQL.ExecContext(ctx, `INSERT INTO link_events (link_id, ip, ua, fingerprint, bytes, suspicious) VALUES (?, ?, ?, ?, ?, ?)`, linkID, ip, ua, fingerprint, bytes, sus)
	return err
}

type LinkStat struct {
	Count   int64
	Active  int64
	Expired int64
}

func (d *DB) GetLinkStat(ctx context.Context) (*LinkStat, error) {
	var s LinkStat
	// 总数
	_ = d.SQL.QueryRowContext(ctx, `SELECT COUNT(1) FROM links`).Scan(&s.Count)
	// 有效
	_ = d.SQL.QueryRowContext(ctx, `SELECT COUNT(1) FROM links WHERE expire_at > CURRENT_TIMESTAMP`).Scan(&s.Active)
	// 过期
	_ = d.SQL.QueryRowContext(ctx, `SELECT COUNT(1) FROM links WHERE expire_at <= CURRENT_TIMESTAMP`).Scan(&s.Expired)
	return &s, nil
}

// 链接列表查询
type LinkItem struct {
	ID              int64
	SourceURL       string
	Filename        string
	Size            int64
	MimeType        string
	Fingerprint     string
	CreatedAt       time.Time
	ExpireAt        time.Time
	DownloadCount   int64
	SuspiciousCount int64
}

func (d *DB) GetLinks(ctx context.Context, limit, offset int) ([]*LinkItem, error) {
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT l.id, l.source_url, f.filename, f.size, f.mime_type, l.fingerprint, l.created_at, l.expire_at,
		       COALESCE(COUNT(e.id), 0) as download_count,
		       COALESCE(SUM(CASE WHEN e.suspicious = 1 THEN 1 ELSE 0 END), 0) as suspicious_count
		FROM links l
		LEFT JOIN files f ON l.source_url = f.source_url
		LEFT JOIN link_events e ON l.id = e.link_id
		GROUP BY l.id
		ORDER BY l.created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*LinkItem
	for rows.Next() {
		item := &LinkItem{}
		err := rows.Scan(&item.ID, &item.SourceURL, &item.Filename, &item.Size, &item.MimeType,
			&item.Fingerprint, &item.CreatedAt, &item.ExpireAt, &item.DownloadCount, &item.SuspiciousCount)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

// 统计查询
type Stats24h struct {
	Hour  string
	Count int64
	Bytes int64
}

func (d *DB) GetStats24h(ctx context.Context) ([]*Stats24h, error) {
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT strftime('%H', at) as hour, COUNT(*) as count, COALESCE(SUM(bytes), 0) as bytes
		FROM link_events 
		WHERE at >= datetime('now', '-24 hours')
		GROUP BY strftime('%H', at)
		ORDER BY hour
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*Stats24h
	for rows.Next() {
		s := &Stats24h{}
		err := rows.Scan(&s.Hour, &s.Count, &s.Bytes)
		if err != nil {
			continue
		}
		stats = append(stats, s)
	}
	return stats, nil
}

type TopSource struct {
	SourceURL string
	Count     int64
	Bytes     int64
}

func (d *DB) GetTopSources(ctx context.Context, limit int) ([]*TopSource, error) {
	rows, err := d.SQL.QueryContext(ctx, `
		SELECT l.source_url, COUNT(e.id) as count, COALESCE(SUM(e.bytes), 0) as bytes
		FROM links l
		LEFT JOIN link_events e ON l.id = e.link_id
		GROUP BY l.source_url
		ORDER BY count DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*TopSource
	for rows.Next() {
		s := &TopSource{}
		err := rows.Scan(&s.SourceURL, &s.Count, &s.Bytes)
		if err != nil {
			continue
		}
		sources = append(sources, s)
	}
	return sources, nil
}
