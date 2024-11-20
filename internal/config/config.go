package config

import (
	"context"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	DisallowedNamespaces []string `env:"DISALLOWED_NAMESPACES"`
}

func Read() (*Config, error) {
	var c Config
	if err := envconfig.Process(context.Background(), &c); err != nil {
		return nil, err
	}

	return &c, nil
}
