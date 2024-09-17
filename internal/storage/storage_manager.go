package storage

import (
	"fmt"
	"sort"
	"sync"
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

	var wg sync.WaitGroup
	for _, storage := range sm.storages {
		wg.Add(1)

		go func(s Storage) {
			defer wg.Done()

			_, err := s.Write(fileName, data)
			if err != nil {
				log.Error("Failed to save file to storage!", "storage", s.Name(), "file", fileName)
			}
		}(storage)
	}
	wg.Wait()

	return fileName
}

func (sm *Manager) Cleanup(retention int) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for _, storage := range sm.storages {
		wg.Add(1)
		go func(s Storage) {
			defer wg.Done()

			files, err := s.List()
			if err != nil {
				log.Error("Cannot get files list from storage!", "storage", s.Name())
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}

			if len(files) <= retention {
				log.Debug("There are no old snapshots to be removed", "storage", s.Name())
				return
			}

			sort.Slice(files, func(i, j int) bool {
				return files[i].ModTime.After(files[j].ModTime)
			})

			for _, file := range files[retention:] {
				err := s.Remove(file.Name)
				if err != nil {
					log.Error("Failed to remove file from storage!", "storage", s.Name(), "file", file.Name)
				}
			}
		}(storage)
	}
	wg.Wait()

	return firstErr
}
