package config

import (
	"cli/internal/common"
	"io"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/caarlos0/env/v10"
	"gopkg.in/yaml.v3"
)

// this needs to be a separate package, so it could be imported by others and be init'd first

// ideally we could create automatic documentation for config / env variables during build
// to differentiate between set to empty value vs not set, use pointer types when applicable

// used for redacting sensitive strings from prints/logs
// this should be used for all token fields
type SensitiveString string

const redactedString = "**REDACTED**"

func (s SensitiveString) String() string {
	if string(s) == "" {
		return ""
	}
	return redactedString
}

func (s SensitiveString) Value() string {
	return string(s)
}

const SealEnvPrefix = "SEAL_"

// boolean env vars will need to be set to a value XXX=1 or XXX=true; otherwise will not detect it
type NpmConfig struct {
	ProdOnlyDeps     bool `yaml:"prod-only"             env:"PROD_ONLY"`    // this affects the output of npm list command; only affects direct deps
	IgnoreExtraneous bool `yaml:"ignore-extraneous"     env:"IGNORE_EXTRA"` // will ignore packages that are marked as extraneous (like `npm i XXX --no-save`)
}

type PnpmConfig struct {
	ProdOnlyDeps bool `yaml:"prod-only"         env:"PROD_ONLY"` // same as npm
}

type MavenConfig struct {
	ProdOnlyDeps bool   `yaml:"prod-only"  env:"PROD_ONLY"`
	CachePath    string `yaml:"cache-path" env:"CACHE_PATH"` // since maven uses global cache, we need to set a new cache folder so we can override packages as we please
}

type GradleConfig struct {
	ProdOnlyDeps bool   `yaml:"prod-only"  env:"PROD_ONLY"`
	HomePath     string `yaml:"home-path" env:"HOME_PATH"` // new target user home path to use a private copy of the gradle cache
}

type PythonConfig struct {
	OnlyBinary bool `yaml:"only-binary" env:"ONLY_BINARY"` // only install whl artifacts, no source artifacts
}

type ComposerConfig struct {
	ProdOnlyDeps bool `yaml:"prod-only" env:"PROD_ONLY"`
}

type BlackDuckConfig struct {
	Url         string          `yaml:"blackduck-url"                       env:"URL"`
	Token       SensitiveString `yaml:"blackduck-token"                     env:"TOKEN"`
	Project     string          `yaml:"blackduck-project-name"              env:"PROJECT"`
	VersionName string          `yaml:"blackduck-project-version-name"      env:"PROJECT_VERSION_NAME"`
}

type DependabotConfig struct {
	Url   string          `yaml:"url"       env:"URL"`
	Token SensitiveString `yaml:"token"     env:"TOKEN"`
	Owner string          `yaml:"owner"     env:"OWNER"`
	Repo  string          `yaml:"repo"      env:"REPO"`
}

type OxConfig struct {
	Url                          string          `yaml:"url"       env:"URL"`
	Token                        SensitiveString `yaml:"token"     env:"TOKEN"`
	Application                  string          `yaml:"application" env:"APPLICATION"`
	ExcludeWhenHighCriticalFixed bool            `yaml:"exclude-when-high-critical-fixed" env:"EXCLUDE_WHEN_HIGH_CRITICAL_FIXED"`
}

type ProjectInfo struct {
	Targets []string `yaml:"targets"` // list of scan targets
}

type JFrogConfig struct {
	Token  SensitiveString `yaml:"token"  env:"AUTH_TOKEN"`
	Host   string          `yaml:"host"   env:"INSTANCE_HOST"`
	Scheme string          `yaml:"scheme"   env:"INSTANCE_SCHEME"`

	Enabled bool `yaml:"enabled"   env:"ENABLED"`

	// repository keys for jfrog, will use defaults from `setDefaults` if not set
	CliRepository   string `yaml:"cli-repo"   env:"CLI_REPO"`
	MavenRepository string `yaml:"maven-repo"   env:"MAVEN_REPO"`
}

type Config struct {
	Token      SensitiveString  `yaml:"token"          env:"TOKEN"`
	Project    string           `yaml:"project"        env:"PROJECT"`
	Npm        NpmConfig        `yaml:"npm"            envPrefix:"NPM_"`
	Pnpm       PnpmConfig       `yaml:"pnpm"           envPrefix:"PNPM_"`
	Maven      MavenConfig      `yaml:"maven"          envPrefix:"MAVEN_"`
	Gradle     GradleConfig     `yaml:"gradle"          envPrefix:"GRADLE_"`
	Python     PythonConfig     `yaml:"python"         envPrefix:"PYTHON_"`
	Composer   ComposerConfig   `yaml:"composer"       envPrefix:"PHPCOMPOSER_"`
	Ox         OxConfig         `yaml:"ox"             envPrefix:"OX_"`
	BlackDuck  BlackDuckConfig  `yaml:"blackduck" envPrefix:"BLACKDUCK_"`
	Dependabot DependabotConfig `yaml:"dependabot" envPrefix:"DEPENDABOT_"`

	JFrog JFrogConfig `yaml:"jfrog" envPrefix:"JFROG_"`

	// the following map deprecates the Project field, but we keep support for backward compatibility and utilizing it for 'caching' the selected project
	ProjectMap map[string]ProjectInfo `yaml:"projects"` // project id is the key - no env override for this

	UseSealedNames bool `yaml:"use-sealed-names" env:"USE_SEALED_NAMES"`
}

var FailedParsingConfYaml = common.NewPrintableError("could not parse configuration")
var FailedParsingEnvVars = common.NewPrintableError("could not parse environment variables")
var InvalidJFrogHost = common.NewPrintableError("invalid JFrog host")
var InvalidJFrogHostScheme = common.NewPrintableError("JFrog host should not contain scheme")

const ConfigFileName = ".seal-config.yml"

type EnvGetter interface {
	Getenv(key string) string
}

type EnvMap map[string]string

type EnvLookupFunc func(string) (string, bool)

func setDefaults(conf *Config) {
	conf.JFrog.CliRepository = "seal-cli"
	conf.JFrog.MavenRepository = "seal-mvn"
	conf.JFrog.Scheme = "https"
}

func New(environment EnvMap) (*Config, error) {
	return Load(strings.NewReader(""), environment)
}

func validate(conf Config) *common.PrintableError {
	if conf.JFrog.Enabled {
		host := conf.JFrog.Host
		u, err := url.Parse(host)
		if err != nil {
			slog.Error("jfrog host could not be parsed", "host", host)
			return InvalidJFrogHost
		}

		if u.Scheme != "" {
			slog.Error("jfrog host contains scheme", "scheme", u.Scheme, "host", host)
			return InvalidJFrogHostScheme
		}
	}

	if conf.Gradle.HomePath != "" {
		if !filepath.IsAbs(conf.Gradle.HomePath) {
			slog.Error("gradle home path must be absolute")
			return common.NewPrintableError("gradle home must be an absolute path")
		}
	}

	return nil
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

	if err := validate(*conf); err != nil {
		slog.Error("config failed validation", "err", err)
		return nil, err
	}

	return conf, nil
}
