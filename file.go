package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/imdario/mergo"
	"github.com/unistack-org/micro/v3/codec"
	"github.com/unistack-org/micro/v3/config"
	rutil "github.com/unistack-org/micro/v3/util/reflect"
)

var (
	DefaultStructTag = "file"
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

	if path == "" {
		return ErrPathNotExist
	}

	c.path = path

	return nil
}

func (c *fileConfig) Load(ctx context.Context) error {
	for _, fn := range c.opts.BeforeLoad {
		if err := fn(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}
	}

	fp, err := os.OpenFile(c.path, os.O_RDONLY, os.FileMode(0400))
	if err != nil && !c.opts.AllowFail {
		return fmt.Errorf("failed to open: %s, error: %w", c.path, ErrPathNotExist)
	} else if err == nil {
		defer fp.Close()
		var buf []byte
		buf, err = ioutil.ReadAll(io.LimitReader(fp, int64(codec.DefaultMaxMsgSize)))
		if err == nil {
			src, err := rutil.Zero(c.opts.Struct)
			if err == nil {
				err = c.opts.Codec.Unmarshal(buf, src)
				if err == nil {
					err = mergo.Merge(c.opts.Struct, src, mergo.WithOverride, mergo.WithTypeCheck, mergo.WithAppendSlice)
				}
			}
		}
		if err != nil && !c.opts.AllowFail {
			return err
		}
	}

	for _, fn := range c.opts.AfterLoad {
		if err := fn(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}
	}

	return nil
}

func (c *fileConfig) Save(ctx context.Context) error {
	for _, fn := range c.opts.BeforeSave {
		if err := fn(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}
	}

	buf, err := c.opts.Codec.Marshal(c.opts.Struct)
	if err == nil {
		var fp *os.File
		fp, err = os.OpenFile(c.path, os.O_RDONLY, os.FileMode(0400))
		if err != nil && c.opts.AllowFail {
			return nil
		} else if err != nil && !c.opts.AllowFail {
			return fmt.Errorf("failed to open: %s, error: %w", c.path, ErrPathNotExist)
		}

		if _, werr := fp.Write(buf); werr == nil {
			err = fp.Close()
		} else {
			err = werr
		}
	}

	if err != nil && !c.opts.AllowFail {
		return err
	}

	for _, fn := range c.opts.AfterSave {
		if err := fn(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}
	}

	return nil
}

func (c *fileConfig) String() string {
	return "file"
}

func (c *fileConfig) Name() string {
	return c.opts.Name
}

func NewConfig(opts ...config.Option) config.Config {
	options := config.NewOptions(opts...)
	if len(options.StructTag) == 0 {
		options.StructTag = DefaultStructTag
	}
	return &fileConfig{opts: options}
}
