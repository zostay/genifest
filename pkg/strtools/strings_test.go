package strtools_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zostay/genifest/pkg/strtools"
)

func TestIndentSpaces(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "    a\n    b\n    c\n", strtools.IndentSpaces(4, "a\nb\nc\n"))
}

func TestMakeMatch(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "**/*.yaml", strtools.MakeMatch(""))
	assert.Equal(t, "**/a.yaml", strtools.MakeMatch("a"))
	assert.Equal(t, "**/b.json", strtools.MakeMatch("b.json"))
}
