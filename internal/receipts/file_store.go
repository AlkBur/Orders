package receipts

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// FileStore хранит файлы документов в отдельной базе files.db
// (receipt_files). Логическая связь с receipts.id из base.db — через
// receipt_id без FK.
type FileStore struct {
	db *sql.DB
}

func NewFileStore(db *sql.DB) *FileStore {
	return &FileStore{db: db}
}

// ListByReceipt возвращает метаданные файлов документа без содержимого
// (file_data не читается), отсортированные по времени загрузки.
func (s *FileStore) ListByReceipt(ctx context.Context, receiptID int64) ([]*ReceiptFile, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, receipt_id, uuid, file_name, mime_type, file_size,
		       created_at, updated_at
		FROM receipt_files
		WHERE receipt_id = ?
		ORDER BY created_at, id
	`, receiptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*ReceiptFile
	for rows.Next() {
		f := &ReceiptFile{}
		if err := rows.Scan(
			&f.ID, &f.ReceiptID, &f.UUID, &f.FileName, &f.MimeType, &f.FileSize,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if files == nil {
		files = []*ReceiptFile{}
	}
	return files, nil
}

// CountByReceipts возвращает количество файлов для каждого документа из
// списка. Выполняется одним запросом (files.db) — без N+1. Документы без
// файлов и отсутствующие в списке в результат не попадают.
func (s *FileStore) CountByReceipts(ctx context.Context, receiptIDs []int64) (map[int64]int, error) {
	counts := make(map[int64]int, len(receiptIDs))
	if len(receiptIDs) == 0 {
		return counts, nil
	}

	placeholders := strings.Repeat("?,", len(receiptIDs))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(receiptIDs))
	for i, id := range receiptIDs {
		args[i] = id
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT receipt_id, COUNT(*)
		FROM receipt_files
		WHERE receipt_id IN (`+placeholders+`)
		GROUP BY receipt_id
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			receiptID int64
			count     int
		)
		if err := rows.Scan(&receiptID, &count); err != nil {
			return nil, err
		}
		counts[receiptID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

// GetByID возвращает файл вместе с содержимым. Оба идентификатора
// проверяются вместе: файл обязан принадлежать документу receiptID,
// иначе возвращается ErrNotFound.
func (s *FileStore) GetByID(ctx context.Context, receiptID, fileID int64) (*ReceiptFile, error) {
	f := &ReceiptFile{}
	err := s.db.QueryRowContext(ctx, `
		SELECT id, receipt_id, uuid, file_name, mime_type, file_size,
		       file_data, created_at, updated_at
		FROM receipt_files
		WHERE id = ? AND receipt_id = ?
	`, fileID, receiptID).Scan(
		&f.ID, &f.ReceiptID, &f.UUID, &f.FileName, &f.MimeType, &f.FileSize,
		&f.Data, &f.CreatedAt, &f.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return f, nil
}

// Upsert атомарно сохраняет файл. Для нового uuid — вставка; для уже
// существующего (receipt_id, uuid) — полная замена содержимого и
// метаданных (file_name, mime_type, file_size, file_data, updated_at).
// Возвращает флаги inserted/updated по результату операции.
func (s *FileStore) Upsert(ctx context.Context, receiptID int64, fileUUID, fileName, mime string, data []byte) (inserted, updated bool, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, false, err
	}
	defer tx.Rollback()

	now := time.Now()

	var existingID int64
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM receipt_files WHERE receipt_id = ? AND uuid = ?`,
		receiptID, fileUUID).Scan(&existingID)
	switch {
	case err == sql.ErrNoRows:
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO receipt_files
				(receipt_id, uuid, file_name, mime_type, file_size, file_data,
				 created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, receiptID, fileUUID, fileName, mime, int64(len(data)), data,
			now.Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
			return false, false, err
		}
		inserted = true
	case err != nil:
		return false, false, err
	default:
		if _, err := tx.ExecContext(ctx, `
			UPDATE receipt_files SET
				file_name = ?, mime_type = ?, file_size = ?, file_data = ?,
				updated_at = ?
			WHERE id = ?
		`, fileName, mime, int64(len(data)), data,
			now.Format(time.RFC3339), existingID); err != nil {
			return false, false, err
		}
		updated = true
	}

	if err := tx.Commit(); err != nil {
		return false, false, err
	}
	return inserted, updated, nil
}
