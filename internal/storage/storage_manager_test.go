package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
	"github.com/vertisan/vault-snapshot-agent/internal/storage"
)

func TestNewStorageManager(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("Create manager with local storage", func(t *testing.T) {
		cfg := &config.StorageConfig{
			Local: config.LocalStorageConfig{
				Path: tempDir,
			},
		}

		manager, err := storage.NewStorageManager(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, manager)
	})
}

func TestManager_SaveFile(t *testing.T) {
	tempDir := t.TempDir()
	cfg := &config.StorageConfig{
		Local: config.LocalStorageConfig{
			Path: tempDir,
		},
	}

	manager, err := storage.NewStorageManager(cfg)
	assert.NoError(t, err)

	t.Run("Save file successfully", func(t *testing.T) {
		data := []byte("test snapshot data")
		fileName := manager.SaveFile(data)

		assert.NotEmpty(t, fileName)
		assert.Contains(t, fileName, "vault-snapshot-")
		assert.Contains(t, fileName, ".snap")

		// Verify file was created
		fullPath := filepath.Join(tempDir, fileName)
		content, err := os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, data, content)
	})
}

func TestManager_Cleanup(t *testing.T) {
	t.Run("Cleanup with retention policy", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := &config.StorageConfig{
			Retention: 2,
			Local: config.LocalStorageConfig{
				Path: tempDir,
			},
		}

		manager, err := storage.NewStorageManager(cfg)
		assert.NoError(t, err)

		// Create 5 snapshot files with different timestamps
		files := []string{
			"vault-snapshot-20240101120000.snap",
			"vault-snapshot-20240102120000.snap",
			"vault-snapshot-20240103120000.snap",
			"vault-snapshot-20240104120000.snap",
			"vault-snapshot-20240105120000.snap",
		}

		for i, fileName := range files {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte("test"), 0644)
			assert.NoError(t, err)

			// Set different modification times
			modTime := time.Now().Add(time.Duration(i) * time.Hour)
			err = os.Chtimes(filePath, modTime, modTime)
			assert.NoError(t, err)
		}

		// Run cleanup
		err = manager.Cleanup(cfg.Retention)
		assert.NoError(t, err)

		// Verify only 2 files remain (the most recent ones)
		entries, err := os.ReadDir(tempDir)
		assert.NoError(t, err)
		assert.Len(t, entries, 2)
	})

	t.Run("Cleanup with no old files", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := &config.StorageConfig{
			Retention: 5,
			Local: config.LocalStorageConfig{
				Path: tempDir,
			},
		}

		manager, err := storage.NewStorageManager(cfg)
		assert.NoError(t, err)

		// Create only 3 files
		for i := 0; i < 3; i++ {
			fileName := filepath.Join(tempDir, "vault-snapshot-2024010"+string(rune('1'+i))+"120000.snap")
			err := os.WriteFile(fileName, []byte("test"), 0644)
			assert.NoError(t, err)
		}

		// Run cleanup
		err = manager.Cleanup(cfg.Retention)
		assert.NoError(t, err)

		// Verify all files remain
		entries, err := os.ReadDir(tempDir)
		assert.NoError(t, err)
		assert.Len(t, entries, 3)
	})

	t.Run("Cleanup with exact retention count", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := &config.StorageConfig{
			Retention: 3,
			Local: config.LocalStorageConfig{
				Path: tempDir,
			},
		}

		manager, err := storage.NewStorageManager(cfg)
		assert.NoError(t, err)

		// Create exactly 3 files
		for i := 0; i < 3; i++ {
			fileName := filepath.Join(tempDir, "vault-snapshot-2024010"+string(rune('1'+i))+"120000.snap")
			err := os.WriteFile(fileName, []byte("test"), 0644)
			assert.NoError(t, err)
		}

		// Run cleanup
		err = manager.Cleanup(cfg.Retention)
		assert.NoError(t, err)

		// Verify all files remain (exactly at retention count)
		entries, err := os.ReadDir(tempDir)
		assert.NoError(t, err)
		assert.Len(t, entries, 3)
	})
}
