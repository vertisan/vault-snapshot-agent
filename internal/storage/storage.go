package storage

import (
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
)

type Storage interface {
	Name() string
	Write(fileName string, data []byte) (string, error)
	Remove(fileName string) error
	List() ([]FileInfo, error)
}

type FileInfo struct {
	Name    string
	ModTime time.Time
	Size    int64
}

func NewStorage(config *config.StorageConfig) ([]Storage, error) {
	var storages []Storage
	var storageNames []string

	if config.Local.Path != "" {
		storage := &LocalStorageDriver{Path: config.Local.Path}
		storages = append(storages, storage)
		storageNames = append(storageNames, storage.Name())
	}

	if len(storages) == 0 {
		log.Fatalf("There are no configured storages!")
	} else {
		log.Infof("Configured storages: %s", strings.Join(storageNames, ", "))
	}

	return storages, nil
}
