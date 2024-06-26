package cfgstr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	cfgstr "github.com/zostay/genifest/pkg/strtools"
)

func TestIndentSpaces(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "    a\n    b\n    c\n", cfgstr.IndentSpaces(4, "a\nb\nc\n"))
}

func TestMakeMatch(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "**/*.yaml", cfgstr.MakeMatch(""))
	assert.Equal(t, "**/a.yaml", cfgstr.MakeMatch("a"))
	assert.Equal(t, "**/b.json", cfgstr.MakeMatch("b.json"))
}
