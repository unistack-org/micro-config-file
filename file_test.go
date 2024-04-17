package file

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"go.unistack.org/micro/v3/codec"
	"go.unistack.org/micro/v3/config"
)

type jsoncodec struct{}

func (*jsoncodec) Marshal(v interface{}, opts ...codec.Option) ([]byte, error) {
	return json.Marshal(v)
}

func (*jsoncodec) Unmarshal(buf []byte, v interface{}, opts ...codec.Option) error {
	return json.Unmarshal(buf, v)
}

func (*jsoncodec) ReadBody(r io.Reader, v interface{}) error {
	return nil
}

func (*jsoncodec) ReadHeader(r io.Reader, m *codec.Message, t codec.MessageType) error {
	return nil
}

func (*jsoncodec) String() string {
	return "json"
}

func (*jsoncodec) Write(w io.Writer, m *codec.Message, v interface{}) error {
	return nil
}

func TestLoadReplace(t *testing.T) {
	type Config struct {
		Key  string
		Pass string
	}
	os.Setenv("PLACEHOLDER", "test")
	cfg := &Config{}
	ctx := context.TODO()
	buf := bytes.NewReader([]byte(`{"key":"val","pass":"${PLACEHOLDER}"}`))
	tr, err := NewEnvTransformer(`(?s)\$\{.*?\}`, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	c := NewConfig(config.Codec(
		&jsoncodec{}),
		config.Struct(cfg),
		Reader(buf),
		Transformer(tr),
	)

	if err := c.Init(); err != nil {
		t.Fatal(err)
	}

	if err := c.Load(ctx); err != nil {
		t.Fatal(err)
	}

	if cfg.Pass != "test" {
		t.Fatalf("not works %#+v\n", cfg)
	}
}
