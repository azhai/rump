// Package config parse and validates command flags.
package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Resource can be either Redis (isRedis) or file.
// URI is either a Redis URI or a file path.
type Resource struct {
	URI      string
	DB       int
	Username string
	Password string
	IsRedis  bool
}

// Config represents the current source and target config.
// Source and target are Resources.
// Verbose enables verbose mode.
// TTL enables keys TTL sync.
type Config struct {
	Source  Resource
	Target  Resource
	Match   string
	Verbose bool
	TTL     bool
}

// exit will exit and print the usage.
// Used in case of errors during flags parse/validate.
func exit(e error) {
	fmt.Println(e)
	flag.PrintDefaults()
	os.Exit(1)
}

// validate makes sure from and to are Redis URIs or file paths,
// and generates the final Config.
func validate(from, to string, verbose, ttl bool) (Config, error) {
	cfg := Config{
		Source: Resource{
			URI: from,
		},
		Target: Resource{
			URI: to,
		},
		Verbose: verbose,
		TTL:     ttl,
	}

	if strings.HasPrefix(from, "redis://") {
		cfg.Source.IsRedis = true
	}

	if strings.HasPrefix(to, "redis://") {
		cfg.Target.IsRedis = true
	}

	// Guard from incorrect usage.
	switch {
	case cfg.Source.URI == "":
		return cfg, fmt.Errorf("from is required")
	case cfg.Target.URI == "":
		return cfg, fmt.Errorf("to is required")
	case !cfg.Source.IsRedis && !cfg.Target.IsRedis:
		return cfg, fmt.Errorf("file-only operations not supported")
	}

	return cfg, nil
}

// Parse parses the command line flags and returns a Config.
func Parse() Config {
	example := "example: redis://127.0.0.1:6379/0 or /tmp/dump.rump"
	from := flag.String("from", "", example)
	to := flag.String("to", "", example)
	match := flag.String("match", "", "match keys")
	verbose := flag.Bool("verbose", false, "optional, verbose output")
	ttl := flag.Bool("ttl", false, "optional, enable ttl sync")

	flag.Parse()

	cfg, err := validate(*from, *to, *verbose, *ttl)
	if err != nil {
		// we exit here instead of returning so that we can show
		// the usage examples in case of an error.
		exit(err)
	}

	cfg.Match = *match

	return cfg
}
