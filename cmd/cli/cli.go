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

	_ "github.com/go-sql-driver/mysql"
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
}

func NewStorage() *Storage {
	return &Storage{}
}

func (s *Storage) Parse(logger log.Logger) (storage.AllStorage, error) {
	if len(s.Storage) != len(s.DSN) {
		return nil, errors.New("must have same number of storage and DSN flags")
	}
	// default storage and DSN pair
	if len(s.Storage) < 1 {
		s.Storage = append(s.Storage, "file")
		s.DSN = append(s.DSN, "db")
	}
	var mdmStorage []storage.AllStorage
	for idx, storage := range s.Storage {
		dsn := s.DSN[idx]
		logger.Info(
			"msg", "storage setup",
			"storage", storage,
		)
		switch storage {
		case "file":
			fileStorage, err := file.New(dsn)
			if err != nil {
				return nil, err
			}
			mdmStorage = append(mdmStorage, fileStorage)
		case "mysql":
			mysqlStorage, err := mysql.New(
				mysql.WithDSN(dsn),
				mysql.WithLogger(logger.With("storage", "mysql")),
			)
			if err != nil {
				return nil, err
			}
			mdmStorage = append(mdmStorage, mysqlStorage)
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
