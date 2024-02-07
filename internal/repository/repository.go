package repository

import (
	"fmt"

	_ "github.com/lib/pq"
	"gitlab.com/syntropynet/amberdm/publisher/osmosis-publisher/pkg/repository"
	"gorm.io/gorm"
)

var _ repository.Repository = (*Repository)(nil)

type Repository struct {
	dbCon *gorm.DB
}

func New(db *gorm.DB) (*Repository, error) {
	ret := &Repository{
		dbCon: db,
	}

	// Create tables for data structures (if table already exists it will not be overwritten)
	err := db.AutoMigrate(&IBCDenom{})
	if err != nil {
		return nil, fmt.Errorf("IBCDenom table migrate error: %w", err)
	}
	err = db.AutoMigrate(&Pool{})
	if err != nil {
		return nil, fmt.Errorf("IBCDenom table migrate error: %w", err)
	}
	err = db.AutoMigrate(&TokenPrice{})
	if err != nil {
		return nil, fmt.Errorf("IBCDenom table migrate error: %w", err)
	}
	return ret, nil
}

func (r *Repository) Close() error {
	return nil
}
