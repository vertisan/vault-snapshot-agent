package storage_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vertisan/vault-snapshot-agent/internal/storage"
)

func TestLocalStorageDriver_Name(t *testing.T) {
	driver := &storage.LocalStorageDriver{Path: "/tmp/test"}
	assert.Equal(t, "local", driver.Name())
}

func TestLocalStorageDriver_Write(t *testing.T) {
	tempDir := t.TempDir()
	driver := &storage.LocalStorageDriver{Path: tempDir}

	t.Run("Write valid file", func(t *testing.T) {
		fileName := "test-file.snap"
		data := []byte("test data")

		fullPath, err := driver.Write(fileName, data)
		assert.NoError(t, err)
		assert.Equal(t, filepath.Join(tempDir, fileName), fullPath)

		// Verify file was written
		content, err := os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, data, content)

		// Verify file permissions (0400 = read-only for owner)
		info, err := os.Stat(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, os.FileMode(0400), info.Mode().Perm())
	})
}

func TestLocalStorageDriver_Remove(t *testing.T) {
	tempDir := t.TempDir()
	driver := &storage.LocalStorageDriver{Path: tempDir}

	t.Run("Remove existing file", func(t *testing.T) {
		fileName := "test-remove.snap"
		filePath := filepath.Join(tempDir, fileName)
		err := os.WriteFile(filePath, []byte("test"), 0644)
		assert.NoError(t, err)

		err = driver.Remove(fileName)
		assert.NoError(t, err)

		// Verify file was removed
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("Remove non-existent file", func(t *testing.T) {
		err := driver.Remove("non-existent.snap")
		assert.Error(t, err)
	})
}

func TestLocalStorageDriver_List(t *testing.T) {
	tempDir := t.TempDir()
	driver := &storage.LocalStorageDriver{Path: tempDir}

	t.Run("List empty directory", func(t *testing.T) {
		files, err := driver.List()
		assert.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("List directory with valid snapshot files", func(t *testing.T) {
		// Create test files
		testFiles := []string{
			"vault-snapshot-20240101120000.snap",
			"vault-snapshot-20240102120000.snap",
		}

		for _, fileName := range testFiles {
			err := os.WriteFile(filepath.Join(tempDir, fileName), []byte("test"), 0644)
			assert.NoError(t, err)
		}

		// Add a delay to ensure different modification times
		time.Sleep(10 * time.Millisecond)

		files, err := driver.List()
		assert.NoError(t, err)
		assert.Len(t, files, 2)

		// Verify file info
		for _, file := range files {
			assert.Contains(t, testFiles, file.Name)
			assert.True(t, file.ModTime.After(time.Time{}))
			assert.Greater(t, file.Size, int64(0))
		}
	})

	t.Run("List directory with non-snapshot files", func(t *testing.T) {
		testDir := t.TempDir()
		driver := &storage.LocalStorageDriver{Path: testDir}

		// Create files that should not be listed
		nonSnapshotFiles := []string{
			"other-file.txt",
			"vault-backup-20240101120000.snap",  // Wrong prefix
			"vault-snapshot-20240101120000.txt", // Wrong extension
		}

		for _, fileName := range nonSnapshotFiles {
			err := os.WriteFile(filepath.Join(testDir, fileName), []byte("test"), 0644)
			assert.NoError(t, err)
		}

		files, err := driver.List()
		assert.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("List directory with subdirectories", func(t *testing.T) {
		testDir := t.TempDir()
		driver := &storage.LocalStorageDriver{Path: testDir}

		// Create a subdirectory that should be ignored
		err := os.Mkdir(filepath.Join(testDir, "subdir"), 0755)
		assert.NoError(t, err)

		// Create a valid snapshot file
		err = os.WriteFile(filepath.Join(testDir, "vault-snapshot-20240101120000.snap"), []byte("test"), 0644)
		assert.NoError(t, err)

		files, err := driver.List()
		assert.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("List non-existent directory", func(t *testing.T) {
		driver := &storage.LocalStorageDriver{Path: "/non/existent/path"}
		files, err := driver.List()
		assert.Error(t, err)
		assert.Nil(t, files)
	})
}
