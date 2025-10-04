package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
	"github.com/vertisan/vault-snapshot-agent/internal/storage"
)

func TestNewStorage(t *testing.T) {
	t.Run("Create storage with local configuration", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Local: config.LocalStorageConfig{
				Path: "/tmp/test",
			},
		}

		storages, err := storage.NewStorage(cfg)
		assert.NoError(t, err)
		assert.Len(t, storages, 1)
		assert.Equal(t, "local", storages[0].Name())
	})
}
