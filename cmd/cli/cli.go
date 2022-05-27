// Package cli contains shared command-line helpers and utilities.
package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/storage"
	"github.com/micromdm/nanomdm/storage/allmulti"
	"github.com/micromdm/nanomdm/storage/file"
	"github.com/micromdm/nanomdm/storage/mysql"
	"github.com/micromdm/nanomdm/storage/postgresql"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type StringAccumulator []string

func (s *StringAccumulator) String() string {
	return strings.Join(*s, ",")
}

func (s *StringAccumulator) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type Storage struct {
	Storage StringAccumulator
	DSN     StringAccumulator
	Options StringAccumulator
}

func NewStorage() *Storage {
	return &Storage{}
}

func (s *Storage) Parse(logger log.Logger) (storage.AllStorage, error) {
	if len(s.Storage) != len(s.DSN) {
		return nil, errors.New("must have same number of storage and DSN flags")
	}
	if len(s.Options) > 0 && len(s.Storage) != len(s.Options) {
		return nil, errors.New("must have same number of storage and storage options flags")
	}
	// default storage and DSN pair
	if len(s.Storage) < 1 {
		s.Storage = append(s.Storage, "file")
		s.DSN = append(s.DSN, "db")
	}
	var mdmStorage []storage.AllStorage
	for idx, storage := range s.Storage {
		dsn := s.DSN[idx]
		options := ""
		if len(s.Options) > 0 {
			options = s.Options[idx]
		}
		logger.Info(
			"msg", "storage setup",
			"storage", storage,
		)
		switch storage {
		case "file":
			fileStorage, err := fileStorageConfig(dsn, options)
			if err != nil {
				return nil, err
			}
			mdmStorage = append(mdmStorage, fileStorage)
		case "mysql":
			mysqlStorage, err := mysqlStorageConfig(dsn, options, logger)
			if err != nil {
				return nil, err
			}
			mdmStorage = append(mdmStorage, mysqlStorage)
		case "postgresql":
			postgresqlStorage, err := postgresqlStorageConfig(dsn, options, logger)
			if err != nil {
				return nil, err
			}
			mdmStorage = append(mdmStorage, postgresqlStorage)
		default:
			return nil, fmt.Errorf("unknown storage: %s", storage)
		}
	}
	if len(mdmStorage) < 1 {
		return nil, errors.New("no storage setup")
	}
	if len(mdmStorage) == 1 {
		return mdmStorage[0], nil
	}
	logger.Info("msg", "storage setup", "storage", "multi-storage", "count", len(mdmStorage))
	return allmulti.New(
		logger.With("component", "multi-storage"),
		mdmStorage...,
	), nil
}

var NoStorageOptions = errors.New("storage backend does not support options, please specify no (or empty) options")

func fileStorageConfig(dsn, options string) (*file.FileStorage, error) {
	if options != "" {
		return nil, NoStorageOptions
	}
	return file.New(dsn)
}

func mysqlStorageConfig(dsn, options string, logger log.Logger) (*mysql.MySQLStorage, error) {
	if options != "" {
		return nil, NoStorageOptions
	}
	opts := []mysql.Option{
		mysql.WithDSN(dsn),
		mysql.WithLogger(logger.With("storage", "mysql")),
	}
	return mysql.New(opts...)
}

func postgresqlStorageConfig(dsn, options string, logger log.Logger) (*postgresql.PgSQLStorage, error) {
	if options != "" {
		return nil, NoStorageOptions
	}
	opts := []postgresql.Option{
		postgresql.WithDSN(dsn),
		postgresql.WithLogger(logger.With("storage", "postgresql")),
	}
	return postgresql.New(opts...)
}
