package config

import (
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var _k = koanf.New(".")

func Read() {
	_ = _k.Load(file.Provider("replik8or.yaml"), yaml.Parser())

	_ = _k.Load(env.ProviderWithValue("REPLIK8OR_", ".", func(s, v string) (string, any) {
		s = strings.ToLower(strings.TrimPrefix(s, "REPLIK8OR_"))
		if strings.Contains(v, ",") {
			return s, strings.Split(v, ",")
		}
		return s, v
	}), nil)
}

func StrSlice(key string) []string {
	return _k.Strings(key)
}
