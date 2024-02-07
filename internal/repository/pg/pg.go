package pg

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(host string, port uint, user string, password string, dbname string) (*gorm.DB, error) {
	dbCon, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname),
		PreferSimpleProtocol: true, // disables implicit prepared statement usage. By default pgx automatically uses the extended protocol
	}), &gorm.Config{})

	if err != nil {
		return nil, err
	}

	return dbCon, nil
}
