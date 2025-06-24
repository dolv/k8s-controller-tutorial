package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	assert.NoError(t, err)

	tests := []struct {
		input    string
		expected string
	}{
		{"~", home},
		{"~/.kube/config", filepath.Join(home, ".kube/config")},
		{"/etc/hosts", "/etc/hosts"},
		{"./relative/path", "./relative/path"},
	}

	if runtime.GOOS == "windows" {
		tests[2].expected = `\etc\hosts` // On Windows, filepaths are joined with backslash
	}

	for _, tt := range tests {
		out, err := ExpandPath(tt.input)
		assert.NoError(t, err)
		assert.Equal(t, tt.expected, out, "input: %s", tt.input)
	}
}
