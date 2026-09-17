package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// ScanRecord 是一次扫描任务的落库快照(issue #111)。
// 与 print_jobs 类似,任务态由后台 goroutine 更新,前端通过 /api/scan/records 查询。
type ScanRecord struct {
	ID         int64
	UserID     int64
	Username   string
	Device     string
	Mode       string
	Resolution int
	Source     string
	Format     string
	Filename   string
	StoredPath string
	SizeBytes  int64
	Status     string
	ErrMsg     string
	CreatedAt  string
	FinishedAt sql.NullString
}

const scanRecordColumns = `s.id, s.user_id, u.username, s.device, s.mode, s.resolution,
	s.source, s.format, s.filename, s.stored_path, s.size_bytes,
	s.status, s.err_msg, s.created_at, s.finished_at`

func scanScanRecord(sc scanner) (ScanRecord, error) {
	var rec ScanRecord
	err := sc.Scan(
		&rec.ID, &rec.UserID, &rec.Username, &rec.Device, &rec.Mode, &rec.Resolution,
		&rec.Source, &rec.Format, &rec.Filename, &rec.StoredPath, &rec.SizeBytes,
		&rec.Status, &rec.ErrMsg, &rec.CreatedAt, &rec.FinishedAt,
	)
	return rec, err
}

type ScanFilter struct {
	Username string
	StartAt  string
	EndAt    string
	Limit    int
}

func InsertScanRecord(ctx context.Context, tx *sql.Tx, rec *ScanRecord) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO scan_records (
		user_id, device, mode, resolution, source, format,
		filename, stored_path, size_bytes, status, err_msg, created_at, finished_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.UserID, rec.Device, rec.Mode, rec.Resolution, rec.Source, rec.Format,
		rec.Filename, rec.StoredPath, rec.SizeBytes, rec.Status, rec.ErrMsg, rec.CreatedAt, rec.FinishedAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateScanStatus 用于扫描完成/失败时更新 status / err_msg / size_bytes / finished_at。
func UpdateScanStatus(ctx context.Context, tx *sql.Tx, id int64, status, errMsg string, sizeBytes int64, finishedAt string) error {
	_, err := tx.ExecContext(ctx, `UPDATE scan_records
		SET status = ?, err_msg = ?, size_bytes = ?, finished_at = ?
		WHERE id = ?`, status, errMsg, sizeBytes, finishedAt, id)
	return err
}

func GetScanRecordByID(ctx context.Context, tx *sql.Tx, id int64) (ScanRecord, error) {
	row := tx.QueryRowContext(ctx, `SELECT `+scanRecordColumns+`
		FROM scan_records s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ?`, id)
	return scanScanRecord(row)
}

func ListScanRecords(ctx context.Context, tx *sql.Tx, filter ScanFilter) ([]ScanRecord, error) {
	args := []interface{}{}
	conds := []string{"1=1"}
	if filter.Username != "" {
		conds = append(conds, "u.username = ?")
		args = append(args, filter.Username)
	}
	if filter.StartAt != "" {
		conds = append(conds, "s.created_at >= ?")
		args = append(args, filter.StartAt)
	}
	if filter.EndAt != "" {
		conds = append(conds, "s.created_at <= ?")
		args = append(args, filter.EndAt)
	}
	query := fmt.Sprintf(`SELECT `+scanRecordColumns+`
		FROM scan_records s
		JOIN users u ON u.id = s.user_id
		WHERE %s
		ORDER BY s.created_at DESC`, strings.Join(conds, " AND "))
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []ScanRecord{}
	for rows.Next() {
		rec, err := scanScanRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func DeleteScanRecord(ctx context.Context, tx *sql.Tx, id int64) error {
	res, err := tx.ExecContext(ctx, "DELETE FROM scan_records WHERE id = ?", id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err == nil && affected == 0 {
		return sql.ErrNoRows
	}
	return err
}
