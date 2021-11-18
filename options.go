package file

import (
	"go.unistack.org/micro/v3/config"
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
