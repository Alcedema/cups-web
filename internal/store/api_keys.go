package store

import (
	"context"
	"database/sql"
)

// APIKey 表示一枚 API 密钥的持久化状态。明文 token 永不入库，只保留 SHA-256
// 十六进制字符串作为查询与鉴权凭据，前缀 Prefix 用于前端展示识别（例如
// cw_ABCD1234 打码后缀）。
type APIKey struct {
	ID         int64
	UserID     int64
	Name       string
	Prefix     string
	TokenHash  string
	CreatedAt  string
	LastUsedAt string
	LastUsedIP string
	ExpiresAt  string
}

type CreateAPIKeyInput struct {
	UserID    int64
	Name      string
	Prefix    string
	TokenHash string
	ExpiresAt string
}

// ErrAPIKeyNotFound 由 GetAPIKeyByHash / DeleteAPIKey 在无匹配行时返回。
// 复用 sql.ErrNoRows 语义，让上游一体化处理。
var ErrAPIKeyNotFound = sql.ErrNoRows

func CreateAPIKey(ctx context.Context, tx *sql.Tx, input CreateAPIKeyInput) (APIKey, error) {
	now := nowUTC()
	res, err := tx.ExecContext(ctx, `INSERT INTO api_keys (
		user_id, name, prefix, token_hash, expires_at, created_at
	) VALUES (?, ?, ?, ?, ?, ?)`,
		input.UserID, input.Name, input.Prefix, input.TokenHash, input.ExpiresAt, now,
	)
	if err != nil {
		return APIKey{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return APIKey{}, err
	}
	return getAPIKeyByID(ctx, tx, id)
}

func ListAPIKeysByUser(ctx context.Context, tx *sql.Tx, userID int64) ([]APIKey, error) {
	rows, err := tx.QueryContext(ctx, `SELECT
		id, user_id, name, prefix, token_hash, created_at,
		last_used_at, last_used_ip, expires_at
		FROM api_keys WHERE user_id = ? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := []APIKey{}
	for rows.Next() {
		k, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// GetAPIKeyByHash 按 token_hash 查找。存在唯一索引保证匹配至多一行。
func GetAPIKeyByHash(ctx context.Context, tx *sql.Tx, hash string) (APIKey, error) {
	row := tx.QueryRowContext(ctx, `SELECT
		id, user_id, name, prefix, token_hash, created_at,
		last_used_at, last_used_ip, expires_at
		FROM api_keys WHERE token_hash = ?`, hash)
	return scanAPIKey(row)
}

// DeleteAPIKey 只删除归属于指定用户的 key。用 user_id 二次约束避免越权：
// 前端传 id 时无法删掉别人名下的 key。
func DeleteAPIKey(ctx context.Context, tx *sql.Tx, id, userID int64) error {
	res, err := tx.ExecContext(ctx, "DELETE FROM api_keys WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

// TouchAPIKey 更新最近使用时间与 IP，鉴权成功时调用。失败不影响放行——
// 它是审计信息，而不是决定权。
func TouchAPIKey(ctx context.Context, tx *sql.Tx, id int64, ip string) error {
	_, err := tx.ExecContext(ctx, `UPDATE api_keys SET last_used_at = ?, last_used_ip = ? WHERE id = ?`,
		nowUTC(), ip, id)
	return err
}

func getAPIKeyByID(ctx context.Context, tx *sql.Tx, id int64) (APIKey, error) {
	row := tx.QueryRowContext(ctx, `SELECT
		id, user_id, name, prefix, token_hash, created_at,
		last_used_at, last_used_ip, expires_at
		FROM api_keys WHERE id = ?`, id)
	return scanAPIKey(row)
}

func scanAPIKey(s scanner) (APIKey, error) {
	var k APIKey
	err := s.Scan(
		&k.ID, &k.UserID, &k.Name, &k.Prefix, &k.TokenHash, &k.CreatedAt,
		&k.LastUsedAt, &k.LastUsedIP, &k.ExpiresAt,
	)
	return k, err
}
