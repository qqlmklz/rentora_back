package repository

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// LogApplicationsSnapshot логирует все заявки (ограничение по строкам) и счётчики по БД.
// Аналог «Request.findAll()»: видно, есть ли хоть одна строка со status = completed.
func (db *DB) LogApplicationsSnapshot(ctx context.Context) {
	var total int64
	if err := db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM applications`).Scan(&total); err != nil {
		log.Printf("[db][applications] count total: %v", err)
		return
	}
	var completedCount int64
	if err := db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM applications WHERE LOWER(TRIM(status)) = LOWER(TRIM($1))
	`, ApplicationStatusCompleted).Scan(&completedCount); err != nil {
		log.Printf("[db][applications] count completed: %v", err)
		return
	}

	const maxRows = 500
	rows, err := db.Pool.Query(ctx, `
		SELECT id, status FROM applications ORDER BY id ASC LIMIT $1
	`, maxRows)
	if err != nil {
		log.Printf("[db][applications] list: %v", err)
		return
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var id int
		var status string
		if err := rows.Scan(&id, &status); err != nil {
			log.Printf("[db][applications] scan: %v", err)
			return
		}
		parts = append(parts, fmt.Sprintf("%d:%q", id, status))
	}
	if err := rows.Err(); err != nil {
		log.Printf("[db][applications] rows: %v", err)
		return
	}

	suffix := ""
	if total > maxRows {
		suffix = fmt.Sprintf(" (показано первых %d из %d)", maxRows, total)
	}
	log.Printf("[db][applications] diagnostics: total=%d status_completed=%d%s snapshot=[%s]",
		total, completedCount, suffix, strings.Join(parts, ", "))

	if total > 0 && completedCount == 0 {
		log.Printf("[db][applications] WARN: ни одной заявки со status=completed — проверьте POST confirm-tenant-expenses и POST complete-owner-request (оба пишут completed в БД)")
	}
}
