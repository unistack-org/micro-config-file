package file

import (
	"io"
	"os"
	"regexp"

	"go.unistack.org/micro/v4/config"
	"golang.org/x/text/transform"
)

type pathKey struct{}

func Path(path string) config.Option {
	return config.SetOption(pathKey{}, path)
}

func LoadPath(path string) config.LoadOption {
	return config.SetLoadOption(pathKey{}, path)
}

func SavePath(path string) config.SaveOption {
	return config.SetSaveOption(pathKey{}, path)
}

func WatchPath(path string) config.WatchOption {
	return config.SetWatchOption(pathKey{}, path)
}

type readerKey struct{}

func Reader(r io.Reader) config.Option {
	return config.SetOption(readerKey{}, r)
}

type transformerKey struct{}

type TransformerFunc func(src []byte, index []int) []byte

func Transformer(t transform.Transformer) config.Option {
	return config.SetOption(transformerKey{}, t)
}

func NewEnvTransformer(rs string, trimLeft, trimRight int) (*EnvTransformer, error) {
	re, err := regexp.Compile(rs)
	if err != nil {
		return nil, err
	}
	return &EnvTransformer{
		Regexp: re,
		TransformerFunc: func(src []byte, index []int) []byte {
			var envKey string
			if len(src) > index[1]-trimRight {
				envKey = string(src[index[0]+trimLeft : index[1]-trimRight])
			}

			if envVal, ok := os.LookupEnv(envKey); ok {
				return []byte(envVal)
			}

			return src[index[0]:index[1]]
		},
	}, nil
}
