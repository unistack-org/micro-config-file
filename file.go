package file // import "go.unistack.org/micro-config-file/v3"

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"

	"dario.cat/mergo"
	"go.unistack.org/micro/v3/config"
	rutil "go.unistack.org/micro/v3/util/reflect"
	"golang.org/x/text/transform"
)

var (
	DefaultStructTag       = "file"
	MaxFileSize      int64 = 1 * 1024 * 1024
)

type fileConfig struct {
	opts        config.Options
	path        string
	reader      io.Reader
	transformer transform.Transformer
}

func (c *fileConfig) Options() config.Options {
	return c.opts
}

func (c *fileConfig) Init(opts ...config.Option) error {
	if err := config.DefaultBeforeInit(c.opts.Context, c); err != nil && !c.opts.AllowFail {
		return err
	}

	for _, o := range opts {
		o(&c.opts)
	}

	if c.opts.Context != nil {
		if v, ok := c.opts.Context.Value(pathKey{}).(string); ok {
			c.path = v
		}
		if v, ok := c.opts.Context.Value(transformerKey{}).(transform.Transformer); ok {
			c.transformer = v
		}
		if v, ok := c.opts.Context.Value(readerKey{}).(io.Reader); ok {
			c.reader = v
		}
	}

	if c.opts.Codec == nil {
		return fmt.Errorf("Codec must be specified")
	}

	if err := config.DefaultAfterInit(c.opts.Context, c); err != nil && !c.opts.AllowFail {
		return err
	}

	return nil
}

func (c *fileConfig) Load(ctx context.Context, opts ...config.LoadOption) error {
	if c.opts.SkipLoad != nil && c.opts.SkipLoad(ctx, c) {
		return nil
	}

	if err := config.DefaultBeforeLoad(ctx, c); err != nil && !c.opts.AllowFail {
		return err
	}

	path := c.path
	transformer := c.transformer
	reader := c.reader

	options := config.NewLoadOptions(opts...)
	if options.Context != nil {
		if v, ok := options.Context.Value(pathKey{}).(string); ok && v != "" {
			path = v
		}
		if v, ok := c.opts.Context.Value(transformerKey{}).(transform.Transformer); ok {
			transformer = v
		}
		if v, ok := c.opts.Context.Value(readerKey{}).(io.Reader); ok {
			reader = v
		}
	}

	var fp io.Reader
	var err error

	if c.path != "" {
		fp, err = os.OpenFile(path, os.O_RDONLY, os.FileMode(0o400))
	} else if c.reader != nil {
		fp = reader
	} else {
		err = fmt.Errorf("Path or Reader must be specified")
	}

	if err != nil {
		if !c.opts.AllowFail {
			if c.path != "" {
				return fmt.Errorf("file load path %s error: %w", path, err)
			} else {
				return fmt.Errorf("file load error: %w", err)
			}
		}
		if err = config.DefaultAfterLoad(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}

		return nil
	}

	if fpc, ok := fp.(io.Closer); ok {
		defer fpc.Close()
	}

	var r io.Reader
	if transformer != nil {
		r = transform.NewReader(fp, c.transformer)
	} else {
		r = fp
	}

	buf, err := io.ReadAll(io.LimitReader(r, MaxFileSize))
	if err != nil {
		if !c.opts.AllowFail {
			return err
		}
		if err = config.DefaultAfterLoad(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}

		return nil
	}

	dst := c.opts.Struct
	if options.Struct != nil {
		dst = options.Struct
	}

	src, err := rutil.Zero(dst)
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
			err = mergo.Merge(dst, src, mopts...)
		}
	}

	if err != nil && !c.opts.AllowFail {
		return err
	}

	if err := config.DefaultAfterLoad(ctx, c); err != nil && !c.opts.AllowFail {
		return err
	}

	return nil
}

func (c *fileConfig) Save(ctx context.Context, opts ...config.SaveOption) error {
	if c.opts.SkipSave != nil && c.opts.SkipSave(ctx, c) {
		return nil
	}

	if err := config.DefaultBeforeSave(ctx, c); err != nil && !c.opts.AllowFail {
		return err
	}

	path := c.path
	options := config.NewSaveOptions(opts...)
	if options.Context != nil {
		if v, ok := options.Context.Value(pathKey{}).(string); ok && v != "" {
			path = v
		}
	}

	dst := c.opts.Struct
	if options.Struct != nil {
		dst = options.Struct
	}

	buf, err := c.opts.Codec.Marshal(dst)
	if err != nil {
		if !c.opts.AllowFail {
			return err
		}
		if err = config.DefaultAfterSave(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}

		return nil
	}

	fp, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, os.FileMode(0o600))
	if err != nil {
		if !c.opts.AllowFail {
			return err
		}
		if err = config.DefaultAfterSave(ctx, c); err != nil && !c.opts.AllowFail {
			return err
		}

		return nil
	}
	defer fp.Close()

	if _, err = fp.Write(buf); err == nil {
		err = fp.Close()
	}

	if err != nil && !c.opts.AllowFail {
		return err
	}

	if err := config.DefaultAfterSave(ctx, c); err != nil && !c.opts.AllowFail {
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

type EnvTransformer struct {
	maxMatchSize    int
	Regexp          *regexp.Regexp
	TransformerFunc TransformerFunc
	overflow        []byte
}

var _ transform.Transformer = (*EnvTransformer)(nil)

// Transform implements golang.org/x/text/transform#Transformer
func (t *EnvTransformer) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	t.maxMatchSize = 1024
	var n int
	// copy any overflow from the last call
	if len(t.overflow) > 0 {
		n, err = fullcopy(dst, t.overflow)
		nDst += n
		if err != nil {
			t.overflow = t.overflow[n:]
			return
		}
		t.overflow = nil
	}
	for _, index := range t.Regexp.FindAllSubmatchIndex(src, -1) {
		// copy everything up to the match
		n, err = fullcopy(dst[nDst:], src[nSrc:index[0]])
		nSrc += n
		nDst += n
		if err != nil {
			return
		}
		// skip the match if it ends at the end the src buffer.
		// it could potentially match more
		if index[1] == len(src) && !atEOF {
			break
		}
		// copy the replacement
		rep := t.TransformerFunc(src, index)
		n, err = fullcopy(dst[nDst:], rep)
		nDst += n
		nSrc = index[1]
		if err != nil {
			t.overflow = rep[n:]
			return
		}
	}
	// if we're at the end, tack on any remaining bytes
	if atEOF {
		n, err = fullcopy(dst[nDst:], src[nSrc:])
		nDst += n
		nSrc += n
		return
	}
	// skip any bytes which exceede the max match size
	if skip := len(src[nSrc:]) - t.maxMatchSize; skip > 0 {
		n, err = fullcopy(dst[nDst:], src[nSrc:nSrc+skip])
		nSrc += n
		nDst += n
		if err != nil {
			return
		}
	}
	err = transform.ErrShortSrc
	return
}

// Reset resets the state and allows a Transformer to be reused.
func (t *EnvTransformer) Reset() {
	t.overflow = nil
}

func fullcopy(dst, src []byte) (n int, err error) {
	n = copy(dst, src)
	if n < len(src) {
		err = transform.ErrShortDst
	}
	return
}
