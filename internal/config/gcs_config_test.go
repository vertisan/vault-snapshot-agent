package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vertisan/vault-snapshot-agent/internal/config"
)

func TestGCSConfiguration(t *testing.T) {
	yamlContent := `
vault:
  addr: "https://127.0.0.1:8200"
  roleId: "test-role-id"
  secretId: "test-secret-id"
  approle: "approle"
storage:
  retention: 5
  gcs:
    bucket: "my-vault-snapshots"
    prefix: "production/snapshots"
`
	tmpFile := "/tmp/test-gcs.yaml"
	err := os.WriteFile(tmpFile, []byte(yamlContent), 0644)
	assert.NoError(t, err)
	defer os.Remove(tmpFile)

	cfg, err := config.LoadConfig(tmpFile)
	assert.NoError(t, err)
	assert.Equal(t, "my-vault-snapshots", cfg.Storage.GCS.Bucket)
	assert.Equal(t, "production/snapshots", cfg.Storage.GCS.Prefix)
	assert.Equal(t, 5, cfg.Storage.Retention)
}
