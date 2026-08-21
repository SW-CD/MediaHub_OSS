package sqlitemigrations

import (
	"sync"

	"github.com/pressly/goose/v3"
)

var registerOnce sync.Once

// Register explicitly registers SQLite Go migrations into Goose.
// It is called when initializing the SQLite repository driver.
func Register() {
	registerOnce.Do(func() {
		goose.AddNamedMigrationContext("02001_migrate_queue_system.go", up02001, down02001)
		goose.AddNamedMigrationContext("03000_entry_indexes.go", up03000, down03000)
		goose.AddNamedMigrationContext("03001_migrate_user_ulids.go", up03001, down03001)
	})
}
