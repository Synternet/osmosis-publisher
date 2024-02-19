package repository

import (
	"fmt"
	"log/slog"

	_ "github.com/lib/pq"
	"github.com/syntropynet/osmosis-publisher/pkg/repository"
	"gorm.io/gorm"
)

var _ repository.Repository = (*Repository)(nil)

type Repository struct {
	logger *slog.Logger
	dbCon  *gorm.DB
}

func New(db *gorm.DB, logger *slog.Logger) (*Repository, error) {
	ret := &Repository{
		logger: logger.With("module", "repository"),
		dbCon:  db,
	}

	// Create tables for data structures (if table already exists it will not be overwritten)
	err := db.AutoMigrate(&IBCDenom{})
	if err != nil {
		return nil, fmt.Errorf("IBCDenom migrate error: %w", err)
	}
	err = db.AutoMigrate(&Pool{})
	if err != nil {
		return nil, fmt.Errorf("Pool migrate error: %w", err)
	}
	err = db.AutoMigrate(&TokenPrice{})
	if err != nil {
		return nil, fmt.Errorf("TokenPrice migrate error: %w", err)
	}
	return ret, nil
}

func (r *Repository) Close() error {
	return nil
}
