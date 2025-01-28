package config

import (
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

var k = koanf.New(".")

func Read() error {
	if err := k.Load(file.Provider("replik8or.yaml"), yaml.Parser()); err != nil {
		return err
	}

	return k.Load(env.Provider("REPLIK8OR_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "REPLIK8OR_"))
	}), nil)
}

func String(key string) string {
	return k.String(key)
}

func StrSlice(key string) []string {
	return k.Strings(key)
}
