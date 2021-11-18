package file // import "go.unistack.org/micro-config-file/v3"

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/imdario/mergo"
	"go.unistack.org/micro/v3/codec"
	"go.unistack.org/micro/v3/config"
	rutil "go.unistack.org/micro/v3/util/reflect"
)

var DefaultStructTag = "file"

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

	if c.opts.Context != nil {
		if v, ok := c.opts.Context.Value(pathKey{}).(string); ok {
			c.path = v
		}
	}

	if c.path == "" {
		err := fmt.Errorf("file path not exists: %v", c.path)
		c.opts.Logger.Error(c.opts.Context, err)
		if !c.opts.AllowFail {
			return err
		}
	}

	return nil
}

func (c *fileConfig) Load(ctx context.Context, opts ...config.LoadOption) error {
	if err := config.DefaultBeforeLoad(ctx, c); err != nil {
		return err
	}

	path := c.path
	options := config.NewLoadOptions(opts...)
	if options.Context != nil {
		if v, ok := options.Context.Value(pathKey{}).(string); ok && v != "" {
			path = v
		}
	}

	fp, err := os.OpenFile(path, os.O_RDONLY, os.FileMode(0400))
	if err != nil {
		c.opts.Logger.Errorf(c.opts.Context, "file load path %s error: %v", path, err)
		if !c.opts.AllowFail {
			return err
		}
		return config.DefaultAfterLoad(ctx, c)
	}

	defer fp.Close()

	buf, err := ioutil.ReadAll(io.LimitReader(fp, int64(codec.DefaultMaxMsgSize)))
	if err != nil {
		c.opts.Logger.Errorf(c.opts.Context, "file load path %s error: %v", path, err)
		if !c.opts.AllowFail {
			return err
		}
		return config.DefaultAfterLoad(ctx, c)
	}

	src, err := rutil.Zero(c.opts.Struct)
	if err == nil {
		err = c.opts.Codec.Unmarshal(buf, src)
		if err == nil {
			options := config.NewLoadOptions(opts...)
			mopts := []func(*mergo.Config){mergo.WithTypeCheck}
			if options.Override {
				mopts = append(mopts, mergo.WithOverride)
			}
			if options.Append {
				mopts = append(mopts, mergo.WithAppendSlice)
			}
			err = mergo.Merge(c.opts.Struct, src, mopts...)
		}
	}

	if err != nil {
		c.opts.Logger.Errorf(c.opts.Context, "file load path %s error: %v", path, err)
		if !c.opts.AllowFail {
			return err
		}
	}

	if err := config.DefaultAfterLoad(ctx, c); err != nil {
		return err
	}

	return nil
}

func (c *fileConfig) Save(ctx context.Context, opts ...config.SaveOption) error {
	if err := config.DefaultBeforeSave(ctx, c); err != nil {
		return err
	}

	path := c.path
	options := config.NewSaveOptions(opts...)
	if options.Context != nil {
		if v, ok := options.Context.Value(pathKey{}).(string); ok && v != "" {
			path = v
		}
	}

	buf, err := c.opts.Codec.Marshal(c.opts.Struct)
	if err != nil {
		c.opts.Logger.Errorf(c.opts.Context, "file save path %s error: %v", path, err)
		if !c.opts.AllowFail {
			return err
		}
		return config.DefaultAfterSave(ctx, c)
	}

	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.FileMode(0600))
	if err != nil {
		c.opts.Logger.Errorf(c.opts.Context, "file save path %s error: %v", path, err)
		if !c.opts.AllowFail {
			return err
		}
		return config.DefaultAfterSave(ctx, c)
	}
	defer fp.Close()

	if _, err = fp.Write(buf); err == nil {
		err = fp.Close()
	}

	if err != nil {
		c.opts.Logger.Errorf(c.opts.Context, "file save path %s error: %v", path, err)
		if !c.opts.AllowFail {
			return err
		}
	}

	if err := config.DefaultAfterSave(ctx, c); err != nil {
		return err
	}

	return nil
}

func (c *fileConfig) String() string {
	return "file"
}

func (c *fileConfig) Name() string {
	return c.opts.Name
}

func (c *fileConfig) Watch(ctx context.Context, opts ...config.WatchOption) (config.Watcher, error) {
	path := c.path
	options := config.NewWatchOptions(opts...)
	if options.Context != nil {
		if v, ok := options.Context.Value(pathKey{}).(string); ok && v != "" {
			path = v
		}
	}

	w := &fileWatcher{
		path:  path,
		opts:  c.opts,
		wopts: config.NewWatchOptions(opts...),
		done:  make(chan struct{}),
		vchan: make(chan map[string]interface{}),
		echan: make(chan error),
	}

	go w.run()

	return w, nil
}

func NewConfig(opts ...config.Option) config.Config {
	options := config.NewOptions(opts...)
	if len(options.StructTag) == 0 {
		options.StructTag = DefaultStructTag
	}
	return &fileConfig{opts: options}
}
