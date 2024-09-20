package util

import (
	"encoding/json"

	cppgtypes "github.com/goplus/llgo/chore/llcppg/types"
	"github.com/goplus/llgo/xtool/env"
)

func GetCppgFromPath(filePath string) (*cppgtypes.Config, error) {
	bytes, err := ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	conf := &cppgtypes.Config{}
	err = json.Unmarshal(bytes, &conf)
	if err != nil {
		return nil, err
	}
	conf.CFlags = env.ExpandEnv(conf.CFlags)
	conf.Libs = env.ExpandEnv(conf.Libs)
	return conf, nil
}
