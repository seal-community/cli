package config

import (
	"cli/internal/common"
	"io"
	"log/slog"
	"strings"

	"github.com/caarlos0/env/v10"
	"gopkg.in/yaml.v3"
)

// this needs to be a separate package, so it could be imported by others and be init'd first

// ideally we could create automatic documentation for config / env variables during build
// to differentiate between set to empty value vs not set, use pointer types when applicable

const SealEnvPrefix = "SEAL_"

// boolean env vars will need to be set to a value XXX=1 or XXX=true; otherwise will not detect it
type NpmConfig struct {
	ProdOnlyDeps     bool `yaml:"prod-only"          env:"PROD_ONLY"`    // this affects the output of npm list command; only affects direct deps
	IgnoreExtraneous bool `yaml:"ignore-extraneous"  env:"IGNORE_EXTRA"` // will ignore packages that are marked as extraneous (like `npm i XXX --no-save`)
}

type PnpmConfig struct {
	ProdOnlyDeps bool `yaml:"prod-only"          env:"PROD_ONLY"` // same as np
}
type Config struct {
	Token   string     `yaml:"token"           env:"TOKEN"`
	Project string     `yaml:"project"         env:"PROJECT"`
	Npm     NpmConfig  `yaml:"npm"             envPrefix:"NPM_"`
	Pnpm    PnpmConfig `yaml:"pnpm"             envPrefix:"PNPM_"`
}

var FailedParsingConfYaml = common.NewPrintableError("could not parse configuration")
var FailedParsingEnvVars = common.NewPrintableError("could not parse environment variables")

const ConfigFileName = ".seal-config.yml"

type EnvGetter interface {
	Getenv(key string) string
}

type EnvMap map[string]string

type EnvLookupFunc func(string) (string, bool)

func setDefaults(conf *Config) {
}

func New(environment EnvMap) (*Config, error) {
	return Load(strings.NewReader(""), environment)
}

func Load(r io.Reader, environment EnvMap) (*Config, error) {
	d := yaml.NewDecoder(r)
	d.KnownFields(true)
	conf := &Config{}

	// init defaults before loading anything
	setDefaults(conf)

	// decode config file
	if err := d.Decode(&conf); err != nil {
		if err == io.EOF {
			slog.Warn("yaml config file is empty")
		} else {
			slog.Error("failed decoding yaml config", "err", err)
			return nil, FailedParsingConfYaml
		}
	}

	// override with environment variables
	// can't use library's default feature because yaml takes precedence over default
	opts := env.Options{Prefix: SealEnvPrefix}
	if environment != nil {
		opts.Environment = environment
	}

	if err := env.ParseWithOptions(conf, opts); err != nil {
		slog.Error("failed decoding env vars", "err", err)
		return nil, FailedParsingEnvVars
	}

	return conf, nil
}
