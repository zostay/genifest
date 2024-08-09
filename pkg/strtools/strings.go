package strtools

import (
	"path/filepath"
	"strings"

	zstrings "github.com/zostay/go-std/strings"

	_ "github.com/zostay/ghost/pkg/secrets/cache"
	_ "github.com/zostay/ghost/pkg/secrets/http"
	_ "github.com/zostay/ghost/pkg/secrets/human"
	_ "github.com/zostay/ghost/pkg/secrets/keepass"
	_ "github.com/zostay/ghost/pkg/secrets/lastpass"
	_ "github.com/zostay/ghost/pkg/secrets/onepassword"
	_ "github.com/zostay/ghost/pkg/secrets/policy"
)

func IndentSpaces(n int, s string) string {
	return zstrings.Indent(s, strings.Repeat(" ", n))
}

func MakeMatch(match string) string {
	if match == "" {
		match = "**/*"
	}

	if len(strings.Split(match, "/")) == 1 {
		match = "**/" + match
	}

	if filepath.Ext(match) == "" {
		match += ".yaml"
	}

	return match
}
