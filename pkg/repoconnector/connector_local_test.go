package repoconnector

import (
	"io"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testdataPath(subpath string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", subpath)
}

func TestLocalConnector_Clone(t *testing.T) {
	tests := []struct {
		name      string
		basePath  string
		expectErr bool
	}{
		{
			name:      "returns worktree for valid directory",
			basePath:  testdataPath("valid-appconfig"),
			expectErr: false,
		},
		{
			name:      "returns worktree even for non-existent directory (error on Open)",
			basePath:  testdataPath("non-existent"),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector := NewLocalConnector(tt.basePath)

			// url と branch は無視される
			wt, err := connector.Clone(t.Context(), "https://example.com/repo.git", "main")

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, wt)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, wt)
			}
		})
	}
}

func TestLocalWorktree_Open(t *testing.T) {
	tests := []struct {
		name        string
		basePath    string
		filePath    string
		expectErr   bool
		expectEmpty bool
	}{
		{
			name:        "opens existing file",
			basePath:    testdataPath("valid-appconfig"),
			filePath:    "appconfig.yaml",
			expectErr:   false,
			expectEmpty: false,
		},
		{
			name:      "returns error for non-existent file",
			basePath:  testdataPath("valid-appconfig"),
			filePath:  "non-existent.yaml",
			expectErr: true,
		},
		{
			name:      "returns error for non-existent directory",
			basePath:  testdataPath("non-existent"),
			filePath:  "appconfig.yaml",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector := NewLocalConnector(tt.basePath)
			wt, err := connector.Clone(t.Context(), "", "")
			require.NoError(t, err)

			f, err := wt.Open(tt.filePath)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, f)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, f)
				defer func() { _ = f.Close() }()

				content, err := io.ReadAll(f)
				assert.NoError(t, err)
				if !tt.expectEmpty {
					assert.NotEmpty(t, content)
				}
			}
		})
	}
}
