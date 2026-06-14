package claude

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    RepoRef
		wantErr bool
	}{
		{"plain", "github.com/chinayin/goxctl-claude", RepoRef{"chinayin", "goxctl-claude"}, false},
		{"short owner/repo (默认 github.com)", "chinayin/goxctl-claude", RepoRef{"chinayin", "goxctl-claude"}, false},
		{"https prefix", "https://github.com/chinayin/goxctl-claude", RepoRef{"chinayin", "goxctl-claude"}, false},
		{"trailing slash", "github.com/chinayin/goxctl-claude/", RepoRef{"chinayin", "goxctl-claude"}, false},
		{"missing repo", "github.com/chinayin", RepoRef{}, true},
		{"too many parts", "github.com/a/b/c", RepoRef{}, true},
		{"empty", "", RepoRef{}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSource(tc.source)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
