package config

import (
	"flag"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct {
	MetricsAddress       string   `mapstructure:"METRICS_ADDR"`
	HealthProbeAddress   string   `mapstructure:"HEALTH_PROBE_ADDR"`
	DisallowedNamespaces []string `mapstructure:"DISALLOWED_NAMESPACES"`
}

var replacer = strings.NewReplacer("-", "_")

func Read() (*Config, error) {
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()

	flag.String("metrics-addr", "0", "The address the metric endpoint binds to. (default 0 = disabled)")
	flag.String("health-probe-addr", "0", "The address the health probe binds to. (default 0 = disabled)")
	flag.String("disallowed-namespaces", "", "A list (comma separated) of namespaces that are disallowed.")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg, decoder); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func decoder(dc *mapstructure.DecoderConfig) {
	dc.MatchName = func(mapKey, fieldName string) bool {
		snakeCase := replacer.Replace(mapKey)
		return strings.ToUpper(snakeCase) == fieldName
	}
}
