package file

import (
	"go.unistack.org/micro/v3/config"
)

type pathKey struct{}

func Path(path string) config.Option {
	return config.SetOption(pathKey{}, path)
}
