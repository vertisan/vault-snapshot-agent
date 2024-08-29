package storage

import (
	"github.com/charmbracelet/log"

	"github.com/vertisan/vault-snapshot-agent/internal/config"
)

type Storage interface {
	SaveFile(fileName string, data []byte) string
}

func NewStorage(config *config.StorageConfig) ([]Storage, error) {
	var storages []Storage

	if config.Local.Path != "" {
		storages = append(storages, &LocalStorageDriver{Path: config.Local.Path})
	}

	if len(storages) == 0 {
		log.Fatalf("There are no configured storages!")
	}

	return storages, nil
}
