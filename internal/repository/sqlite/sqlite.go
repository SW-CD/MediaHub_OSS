package sqlite

import (
	"database/sql"

	"github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

type SQLiteRepository struct {
	DB      *sql.DB
	Cache   *cache.Cache
	Builder squirrel.StatementBuilderType // SQL Query Builder
	Logger  *logrus.Logger
}

func NewRepository(path string) (SQLiteRepository, error) {
	return SQLiteRepository{}, nil
}
