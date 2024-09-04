package storage

import (
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
)

type Manager struct {
	storages []Storage
}

func NewStorageManager(config *config.StorageConfig) (*Manager, error) {
	storages, err := NewStorage(config)
	if err != nil {
		return nil, err
	}
	return &Manager{storages: storages}, nil
}

func (sm *Manager) SaveFile(data []byte) string {
	t := time.Now()
	fileName := fmt.Sprintf("vault-snapshot-%s.snap", t.Format("20060102150405"))

	for _, storage := range sm.storages {
		_, err := storage.SaveFile(fileName, data)
		if err != nil {
			log.Fatal("Cannot save file in storage:", "err", err)
		}
	}

	return fileName
}
