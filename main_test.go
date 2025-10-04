package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	assert.NotEmpty(t, version)
	// Default version should be "dev" when not built with ldflags
	assert.Equal(t, "dev", version)
}
