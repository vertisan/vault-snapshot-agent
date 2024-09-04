package storage

import (
	"errors"

	"github.com/vertisan/vault-snapshot-agent/internal/config"
)

type Storage interface {
	SaveFile(fileName string, data []byte) (string, error)
}

func NewStorage(config *config.StorageConfig) ([]Storage, error) {
	var storages []Storage

	if config.Local.Path != "" {
		storages = append(storages, &LocalStorageDriver{Path: config.Local.Path})
	}

	if len(storages) == 0 {
		return nil, errors.New("there are no configured storages")
	}

	return storages, nil
}
