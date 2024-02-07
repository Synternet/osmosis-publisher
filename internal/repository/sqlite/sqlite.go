package sqlite

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// New opens an SQL database.
// In case in-memory DB is needed(e.g. testing), "file::memory:?cache=shared" can be used instead of a database filename.
func New(dbname string) (*gorm.DB, error) {
	dbCon, err := gorm.Open(sqlite.Open(dbname), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	return dbCon, nil
}
