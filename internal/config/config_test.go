package config

import (
	"flag"
	"os"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestRead(t *testing.T) {
	expected := &Config{
		MetricsAddress:       "testing-metrics-addr",
		HealthProbeAddress:   "testing-health-probe-addr",
		DisallowedNamespaces: []string{"testing-foo", "testing-bar"},
	}

	t.Run("environment variables", func(t *testing.T) {
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		t.Setenv("METRICS_ADDR", expected.MetricsAddress)
		t.Setenv("HEALTH_PROBE_ADDR", expected.HealthProbeAddress)
		t.Setenv("DISALLOWED_NAMESPACES", strings.Join(expected.DisallowedNamespaces, ","))

		actual, err := Read()

		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("flags", func(t *testing.T) {
		pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

		os.Args = []string{
			"", // first arg must be empty
			"--metrics-addr", expected.MetricsAddress,
			"--health-probe-addr", expected.HealthProbeAddress,
			"--disallowed-namespaces", strings.Join(expected.DisallowedNamespaces, ","),
		}

		actual, err := Read()

		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	})
}
