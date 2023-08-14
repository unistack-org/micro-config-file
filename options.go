package file

import (
	"go.unistack.org/micro/v4/options"
)

type pathKey struct{}

func Path(path string) options.Option {
	return options.ContextOption(pathKey{}, path)
}
