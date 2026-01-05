// filepath: internal/repository/recovery_repo.go
package repository

import (
	"fmt"
	"mediahub/internal/logging"
)

// FixZombieEntries scans all databases for entries stuck in 'processing' status
// and updates them to 'error'. It returns the total number of fixed entries.
func (s *Repository) FixZombieEntries() (int, error) {
	databases, err := s.GetDatabases()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve databases: %w", err)
	}

	totalFixed := 0

	for _, db := range databases {
		// Security check on table name (defensive coding)
		if !SafeNameRegex.MatchString(db.Name) {
			logging.Log.Warnf("Skipping potentially unsafe table name during recovery: %s", db.Name)
			continue
		}

		tableName := fmt.Sprintf("entries_%s", db.Name)

		// Execute Update
		query := fmt.Sprintf("UPDATE \"%s\" SET status = 'error' WHERE status = 'processing'", tableName)
		result, err := s.DB.Exec(query)
		if err != nil {
			logging.Log.Errorf("Failed to update table %s: %v", tableName, err)
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			logging.Log.Infof("Fixed %d zombie entries in database '%s'", rowsAffected, db.Name)
			totalFixed += int(rowsAffected)
		}
	}

	return totalFixed, nil
}
