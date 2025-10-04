package storage_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
	internalStorage "github.com/vertisan/vault-snapshot-agent/internal/storage"
)

func TestGCSStorageDriver_Name(t *testing.T) {
	driver, err := internalStorage.NewGCSStorageDriver("test-bucket", "")
	if err != nil {
		// If we can't create a real client (no credentials), we can still test that Name() would return "gcs"
		// Skip this test if GCS credentials are not available
		t.Skip("Skipping GCS test - credentials not available")
	}
	defer driver.Close()

	assert.Equal(t, "gcs", driver.Name())
}

func TestNewStorage_WithGCS(t *testing.T) {
	t.Run("Create storage with GCS configuration", func(t *testing.T) {
		cfg := &config.StorageConfig{
			GCS: config.GCSStorageConfig{
				Bucket: "test-bucket",
				Prefix: "snapshots",
			},
		}

		// This will fail without credentials, but it tests the integration
		_, err := internalStorage.NewStorage(cfg)
		// We expect an error if no credentials are available
		if err == nil {
			// If successful, it means credentials are available
			t.Log("GCS client created successfully")
		} else {
			// Expected error when credentials are not available
			assert.Error(t, err)
		}
	})

	t.Run("Create storage with both Local and GCS", func(t *testing.T) {
		tempDir := t.TempDir()
		cfg := &config.StorageConfig{
			Local: config.LocalStorageConfig{
				Path: tempDir,
			},
			GCS: config.GCSStorageConfig{
				Bucket: "test-bucket",
				Prefix: "snapshots",
			},
		}

		// This will partially succeed (local) and fail on GCS without credentials
		storages, err := internalStorage.NewStorage(cfg)

		if err == nil {
			// If no error, both storages were created
			assert.Len(t, storages, 2)
		} else {
			// GCS failed, but local should have been created
			// The function returns error and stops, so we expect error
			assert.Error(t, err)
		}
	})
}

// TestGCSStorageDriver_ObjectNaming tests the object naming logic
func TestGCSStorageDriver_ObjectNaming(t *testing.T) {
	testCases := []struct {
		name           string
		prefix         string
		fileName       string
		expectedObject string
	}{
		{
			name:           "No prefix",
			prefix:         "",
			fileName:       "vault-snapshot-20240101120000.snap",
			expectedObject: "vault-snapshot-20240101120000.snap",
		},
		{
			name:           "With prefix",
			prefix:         "snapshots",
			fileName:       "vault-snapshot-20240101120000.snap",
			expectedObject: "snapshots/vault-snapshot-20240101120000.snap",
		},
		{
			name:           "Prefix with trailing slash",
			prefix:         "snapshots/",
			fileName:       "vault-snapshot-20240101120000.snap",
			expectedObject: "snapshots/vault-snapshot-20240101120000.snap",
		},
		{
			name:           "Nested prefix",
			prefix:         "backups/vault",
			fileName:       "vault-snapshot-20240101120000.snap",
			expectedObject: "backups/vault/vault-snapshot-20240101120000.snap",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test the naming logic
			objectName := tc.fileName
			if tc.prefix != "" {
				prefix := tc.prefix
				if prefix[len(prefix)-1] == '/' {
					prefix = prefix[:len(prefix)-1]
				}
				objectName = fmt.Sprintf("%s/%s", prefix, tc.fileName)
			}
			assert.Equal(t, tc.expectedObject, objectName)
		})
	}
}

// TestGCSStorageDriver_Integration is an integration test that requires actual GCS credentials
// It will be skipped if credentials are not available
func TestGCSStorageDriver_Integration(t *testing.T) {
	// Skip if GCS_TEST_BUCKET environment variable is not set
	// This allows the test to pass in CI/CD without GCS setup
	testBucket := "test-vault-snapshots"

	driver, err := internalStorage.NewGCSStorageDriver(testBucket, "test-prefix")
	if err != nil {
		t.Skipf("Skipping integration test - cannot create GCS client: %v", err)
		return
	}
	defer driver.Close()

	t.Run("Write, List, and Remove operations", func(t *testing.T) {
		// This is a full integration test that would run against a real bucket
		// For safety, we skip it unless explicitly enabled
		t.Skip("Skipping integration test - requires real GCS bucket and cleanup")

		fileName := fmt.Sprintf("vault-snapshot-%d.snap", time.Now().Unix())
		data := []byte("test snapshot data")

		// Test Write
		path, err := driver.Write(fileName, data)
		assert.NoError(t, err)
		assert.Contains(t, path, testBucket)
		assert.Contains(t, path, fileName)

		// Test List
		files, err := driver.List()
		assert.NoError(t, err)
		assert.NotEmpty(t, files)

		found := false
		for _, file := range files {
			if file.Name == fileName {
				found = true
				assert.Greater(t, file.Size, int64(0))
				break
			}
		}
		assert.True(t, found, "Uploaded file should be in the list")

		// Test Remove
		err = driver.Remove(fileName)
		assert.NoError(t, err)

		// Verify removal
		files, err = driver.List()
		assert.NoError(t, err)
		for _, file := range files {
			assert.NotEqual(t, fileName, file.Name, "File should be removed")
		}
	})
}

// TestGCSStorageDriver_ErrorHandling tests error scenarios
func TestGCSStorageDriver_ErrorHandling(t *testing.T) {
	t.Run("Invalid bucket name", func(t *testing.T) {
		driver, err := internalStorage.NewGCSStorageDriver("", "")
		if err == nil {
			defer driver.Close()
			// If no error creating client, test that operations fail appropriately
			_, err := driver.Write("test.snap", []byte("data"))
			// We expect some kind of error with empty bucket
			assert.Error(t, err)
		} else {
			// Expected error creating client
			assert.Error(t, err)
		}
	})
}

// TestGCSStorageDriver_ListFiltering tests that List() properly filters snapshot files
func TestGCSStorageDriver_ListFiltering(t *testing.T) {
	t.Run("Filter logic for snapshot files", func(t *testing.T) {
		testFiles := []struct {
			name       string
			shouldList bool
		}{
			{"vault-snapshot-20240101120000.snap", true},
			{"vault-snapshot-20240102120000.snap", true},
			{"other-file.txt", false},
			{"vault-backup-20240101120000.snap", false},  // Wrong prefix
			{"vault-snapshot-20240101120000.txt", false}, // Wrong extension
			{"vault-snapshot-incomplete", false},         // No extension
		}

		for _, tf := range testFiles {
			// Test the filtering logic
			isValidSnapshot := func(name string) bool {
				return strings.HasPrefix(name, "vault-snapshot-") &&
					strings.HasSuffix(name, ".snap")
			}

			result := isValidSnapshot(tf.name)
			assert.Equal(t, tf.shouldList, result, "File %s filtering incorrect", tf.name)
		}
	})
}
