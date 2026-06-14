package claude

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadManifest_NotFound(t *testing.T) {
	_, err := LoadManifest(filepath.Join(t.TempDir(), "missing.yaml"))
	require.ErrorIs(t, err, ErrManifestNotFound)
}

func TestSaveManifest_LoadManifest_RoundTrip(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), ManifestFile)
	in := &Manifest{
		Source:  "github.com/chinayin/goxctl-claude",
		Version: "v1.0.0",
		Paths:   []string{"steering/"},
		Target:  ".kiro/steering",
	}

	// Act
	require.NoError(t, SaveManifest(path, in))
	out, err := LoadManifest(path)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, in, out)
}

func TestLoadManifest_DefaultTarget(t *testing.T) {
	// Arrange：Target 留空
	path := filepath.Join(t.TempDir(), ManifestFile)
	require.NoError(t, SaveManifest(path, &Manifest{Source: "x", Version: "v1", Paths: []string{"steering/"}}))

	// Act
	out, err := LoadManifest(path)

	// Assert：缺省回落到 DefaultTarget
	require.NoError(t, err)
	assert.Equal(t, DefaultTarget, out.Target)
}
