package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"

	"go.unistack.org/micro/v3/codec"
	"go.unistack.org/micro/v3/config"
	"go.unistack.org/micro/v3/util/jitter"
	rutil "go.unistack.org/micro/v3/util/reflect"
)

type fileWatcher struct {
	path  string
	opts  config.Options
	wopts config.WatchOptions
	done  chan struct{}
	vchan chan map[string]interface{}
	echan chan error
}

func (w *fileWatcher) run() {
	ticker := jitter.NewTicker(w.wopts.MinInterval, w.wopts.MaxInterval)
	defer ticker.Stop()

	src := w.opts.Struct
	if w.wopts.Struct != nil {
		src = w.wopts.Struct
	}

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
			dst, err := rutil.Zero(src)
			if err == nil {
				var fp *os.File
				if fp, err = os.OpenFile(w.path, os.O_RDONLY, os.FileMode(0400)); err != nil {
					w.echan <- fmt.Errorf("failed to open: %s, error: %w", w.path, err)
					return
				}
				var buf []byte
				buf, err = ioutil.ReadAll(io.LimitReader(fp, int64(codec.DefaultMaxMsgSize)))
				if err == nil {
					err = w.opts.Codec.Unmarshal(buf, dst)
				}
				if err != nil {
					_ = fp.Close()
					w.echan <- err
					return
				}
				err = fp.Close()
			}
			if err != nil {
				w.echan <- err
				return
			}
			srcmp, err := rutil.StructFieldsMap(src)
			if err != nil {
				w.echan <- err
				return
			}
			dstmp, err := rutil.StructFieldsMap(dst)
			if err != nil {
				w.echan <- err
				return
			}
			for sk, sv := range srcmp {
				if reflect.DeepEqual(dstmp[sk], sv) {
					delete(dstmp, sk)
				}
			}
			if len(dstmp) > 0 {
				w.vchan <- dstmp
				src = dst
			}
		}
	}
}

func (w *fileWatcher) Next() (map[string]interface{}, error) {
	select {
	case <-w.done:
		break
	case err := <-w.echan:
		return nil, err
	case v, ok := <-w.vchan:
		if !ok {
			break
		}
		return v, nil
	}
	return nil, config.ErrWatcherStopped
}

func (w *fileWatcher) Stop() error {
	close(w.done)
	return nil
}
