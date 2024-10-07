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
	ProdOnlyDeps       bool `yaml:"prod-only"             env:"PROD_ONLY"`            // this affects the output of npm list command; only affects direct deps
	IgnoreExtraneous   bool `yaml:"ignore-extraneous"     env:"IGNORE_EXTRA"`         // will ignore packages that are marked as extraneous (like `npm i XXX --no-save`)
	UpdatePackageNames bool `yaml:"update-package-names"  env:"UPDATE_PACKAGE_NAMES"` // will update lock file so that fixed packages will have our name
}

type PnpmConfig struct {
	ProdOnlyDeps bool `yaml:"prod-only"         env:"PROD_ONLY"` // same as npm
}

type MavenConfig struct {
	ProdOnlyDeps bool   `yaml:"prod-only"  env:"PROD_ONLY"`
	CachePath    string `yaml:"cache-path" env:"CACHE_PATH"` // since maven uses global cache, we need to set a new cache folder so we can override packages as we please
}

type PythonConfig struct {
	OnlyBinary bool `yaml:"only-binary" env:"ONLY_BINARY"` // only install whl artifacts, no source artifacts
}

type ComposerConfig struct {
	ProdOnlyDeps bool `yaml:"prod-only" env:"PROD_ONLY"`
}

type BlackDuckConfig struct {
	Url         string `yaml:"blackduck-url"                       env:"URL"`
	Token       string `yaml:"blackduck-token"                     env:"TOKEN"`
	Project     string `yaml:"blackduck-project-name"              env:"PROJECT"`
	VersionName string `yaml:"blackduck-project-version-name"      env:"PROJECT_VERSION_NAME"`
}

type ProjectInfo struct {
	Targets []string `yaml:"targets"` // list of scan targets
}

type Config struct {
	Token    string         `yaml:"token"          env:"TOKEN"`
	Project  string         `yaml:"project"        env:"PROJECT"`
	Npm      NpmConfig      `yaml:"npm"            envPrefix:"NPM_"`
	Pnpm     PnpmConfig     `yaml:"pnpm"           envPrefix:"PNPM_"`
	Maven    MavenConfig    `yaml:"maven"          envPrefix:"MAVEN_"`
	Python   PythonConfig   `yaml:"python"         envPrefix:"PYTHON_"`
	Composer ComposerConfig `yaml:"composer"       envPrefix:"PHPCOMPOSER_"`

	BlackDuck BlackDuckConfig `yaml:"blackduck" envPrefix:"BLACKDUCK_"`

	// the following map deprecates the Project field, but we keep support for backward compatibility and utilizing it for 'caching' the selected project
	ProjectMap map[string]ProjectInfo `yaml:"projects"` // project id is the key - no env override for this
}

var FailedParsingConfYaml = common.NewPrintableError("could not parse configuration")
var FailedParsingEnvVars = common.NewPrintableError("could not parse environment variables")

const ConfigFileName = ".seal-config.yml"

type EnvGetter interface {
	Getenv(key string) string
}

type EnvMap map[string]string

type EnvLookupFunc func(string) (string, bool)

func setDefaults(conf *Config) {}

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
