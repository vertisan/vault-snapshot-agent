package storage

import (
	"fmt"
	"sort"
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
		storage.Write(fileName, data)
	}

	return fileName
}

func (sm *Manager) Cleanup(retention int) error {
	for _, storage := range sm.storages {
		files, err := storage.List()
		if err != nil {
			log.Error("Cannot get files list from storage!", "storage")
			return err
		}

		if len(files) <= retention {
			log.Debug("There are no old snapshots to be removed")
			return nil
		}

		sort.Slice(files, func(i, j int) bool {
			return files[i].ModTime.After(files[j].ModTime)
		})

		files = files[retention:]

		for _, file := range files {
			err := storage.Remove(file.Name)
			if err != nil {
				log.Error("Failed to remove file from storage!", "storage", storage.Name(), "file", file.Name)
			}
		}
	}

	return nil
}
