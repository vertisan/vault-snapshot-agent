package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
)

func TestLoadConfig(t *testing.T) {
	// Setup a temporary directory for test files
	tempDir := t.TempDir()

	// Valid configuration content
	validConfigContent := `
vault:
  addr: "https://127.0.0.1:8200"
  roleId: "test-role-id"
  secretId: "test-secret-id"
  approle: "test-approle"
storage:
  local:
    path: "/tmp/vault-backup"
`

	// Invalid configuration content
	invalidConfigContent := `
abc: 123
`

	// Create a valid configuration file
	validConfigPath := filepath.Join(tempDir, "valid-config.yaml")
	err := os.WriteFile(validConfigPath, []byte(validConfigContent), 0644)
	assert.NoError(t, err)

	// Create an invalid configuration file
	invalidConfigPath := filepath.Join(tempDir, "invalid-config.yaml")
	err = os.WriteFile(invalidConfigPath, []byte(invalidConfigContent), 0644)
	assert.NoError(t, err)

	t.Run("Load valid configuration file", func(t *testing.T) {
		configData, err := config.LoadConfig(validConfigPath)
		assert.NoError(t, err)
		assert.NotNil(t, configData)
		assert.Equal(t, "https://127.0.0.1:8200", configData.Vault.Address)
		assert.Equal(t, "test-role-id", configData.Vault.RoleId)
		assert.Equal(t, "test-secret-id", configData.Vault.SecretId)
		assert.Equal(t, "test-approle", configData.Vault.Approle)
		assert.Equal(t, "/tmp/vault-backup", configData.Storage.Local.Path)
	})

	t.Run("Load non-existent configuration file", func(t *testing.T) {
		_, err := config.LoadConfig(filepath.Join(tempDir, "non-existent.yaml"))
		assert.Error(t, err)
	})

	t.Run("Load invalid configuration file", func(t *testing.T) {
		_, err := config.LoadConfig(invalidConfigPath)
		assert.Error(t, err)
	})
}
