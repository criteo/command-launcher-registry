package manifest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize_JSONAndYAMLMatch(t *testing.T) {
	t.Parallel()

	jsonManifest := `{"commands":[{"name":"deploy","description":"Run deploy"}]}`
	yamlManifest := `
commands:
  - name: deploy
    description: Run deploy
`

	jsonCanonical, jsonDigest, err := Normalize(jsonManifest)
	require.NoError(t, err)

	yamlCanonical, yamlDigest, err := Normalize(yamlManifest)
	require.NoError(t, err)

	assert.Equal(t, string(jsonCanonical), string(yamlCanonical))
	assert.Equal(t, jsonDigest, yamlDigest)
	assert.Equal(t, `{"commands":[{"description":"Run deploy","name":"deploy"}]}`, string(jsonCanonical))
}

func TestNormalize_RejectsNonStringKeys(t *testing.T) {
	t.Parallel()

	_, _, err := Normalize("? [1, 2]\n: deploy\n")
	require.Error(t, err)
}
