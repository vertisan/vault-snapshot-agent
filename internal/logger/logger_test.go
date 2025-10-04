package logger_test

import (
	"testing"

	"github.com/vertisan/vault-snapshot-agent/internal/logger"
)

func TestNewLogger(t *testing.T) {
	// Test that NewLogger doesn't panic
	logger.NewLogger()
	// If we get here without panic, the test passes
}
