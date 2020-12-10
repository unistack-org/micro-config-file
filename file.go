package file

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"

	"github.com/unistack-org/micro/v3/codec"
	"github.com/unistack-org/micro/v3/config"
)

var (
	DefaultStructTag = "file"
	ErrInvalidStruct = errors.New("invalid struct specified")
	ErrPathNotExist  = errors.New("path is not exist")
)

type fileConfig struct {
	opts config.Options
	path string
}

func (c *fileConfig) Options() config.Options {
	return c.opts
}

func (c *fileConfig) Init(opts ...config.Option) error {
	for _, o := range opts {
		o(&c.opts)
	}

	path := ""

	if c.opts.Context != nil {
		if v, ok := c.opts.Context.Value(pathKey{}).(string); ok {
			path = v
		}
	}

	c.path = path

	return nil
}

func (c *fileConfig) Load(ctx context.Context) error {
	fp, err := os.OpenFile(c.path, os.O_RDONLY, os.FileMode(0400))
	if err != nil {
		return ErrPathNotExist
	}
	defer fp.Close()

	buf, err := ioutil.ReadAll(io.LimitReader(fp, int64(codec.DefaultMaxMsgSize)))
	if err != nil {
		return err
	}

	return c.opts.Codec.Unmarshal(buf, c.opts.Struct)
}

func (c *fileConfig) Save(ctx context.Context) error {
	return nil
}

func (c *fileConfig) String() string {
	return "file"
}

func NewConfig(opts ...config.Option) config.Config {
	options := config.NewOptions(opts...)
	if len(options.StructTag) == 0 {
		options.StructTag = DefaultStructTag
	}
	return &fileConfig{opts: options}
}
