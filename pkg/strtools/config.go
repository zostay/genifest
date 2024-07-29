package strtools

import "github.com/pelletier/go-toml/v2"

// Tomlize will convert the given object to a TOML string.
func Tomlize(o any) (string, error) {
	bs, err := toml.Marshal(o)
	if err != nil {
		return "", err
	}

	return string(bs), nil
}
