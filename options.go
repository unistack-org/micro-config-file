package file

import (
	"io"
	"os"
	"regexp"

	"go.unistack.org/micro/v4/options"
	"golang.org/x/text/transform"
)

type pathKey struct{}

func Path(path string) options.Option {
	return options.ContextOption(pathKey{}, path)
}

type readerKey struct{}

func Reader(r io.Reader) options.Option {
	return options.ContextOption(readerKey{}, r)
}

type transformerKey struct{}

type TransformerFunc func(src []byte, index []int) []byte

func Transformer(t transform.Transformer) options.Option {
	return options.ContextOption(transformerKey{}, t)
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
