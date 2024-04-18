package actions

import (
	"cli/internal/common"
	"io"
	"log/slog"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/go-playground/validator/v10"

	"gopkg.in/yaml.v3"
)

const SchemaVersion = "0.1.0"

const Iso8601FormatLayout = "2006-01-02T15:04:05Z" // this is a hardcoded time so the the package knows how to parse the layout

// we wrap time.Time to allow custom marshaling with our own format
type IsoTime struct {
	time.Time
}

func (isot IsoTime) MarshalYAML() (interface{}, error) {
	return isot.Time.Format(Iso8601FormatLayout), nil
}

type MetaSection struct {
	SchemaVersion string  `yaml:"schema-version" validate:"required"`
	CreatedOn     IsoTime `yaml:"created-on" validate:"required"`
	CliVersion    string  `yaml:"cli-version" validate:"required"`
}

type Override struct {
	Library string `yaml:"from,omitempty"`
	Version string `yaml:"use" validate:"required"`
}

// NOTE: marshalled sorting will not work as is, due to go's implementation of maps; could cause diffs in output for effectively identical data
type VersionOverrideMap map[string]Override
type LibraryOverrideMap map[string]VersionOverrideMap

type ProjectManagerSection struct {
	Ecosystem string `yaml:"ecosystem" validate:"required"` // currently only node is supported
	Name      string `yaml:"name" validate:"required"`      // like yarn, pnpm, etc
	Version   string `yaml:"version" validate:"required"`
}

type ProjectSection struct {
	Targets   []string              `yaml:"targets" validate:"required,min=1"`
	Manager   ProjectManagerSection `yaml:"manager" validate:"required"`
	Overrides LibraryOverrideMap    `yaml:"overrides" validate:"required,dive,min=1,dive"` // "dive" instructs validation to go 1 layer inside the type; requires twice since map of maps
}

type ActionsFile struct {
	Meta     MetaSection               `yaml:"meta" validate:"required"`
	Projects map[string]ProjectSection `yaml:"projects" validate:"required,min=1,max=1,dive"`
}

var FailedParsingActionYaml = common.NewPrintableError("failed to parse actions file")
var FailedParsingActionYamlEmpty = common.NewPrintableError("empty actions file format")
var FailedParsingActionYamlInvalid = common.NewPrintableError("invalid actions file format")

const ActionFileName = ".seal-actions.yml"

func New() *ActionsFile {
	af := &ActionsFile{}

	// init basic meta fields
	af.Meta.CliVersion = common.CliVersion
	af.Meta.SchemaVersion = SchemaVersion
	af.Meta.CreatedOn = IsoTime{time.Now().UTC()}

	return af
}

func Load(r io.Reader) (*ActionsFile, error) {
	d := yaml.NewDecoder(r)
	d.KnownFields(true)
	actions := &ActionsFile{}

	// decode actions file
	if err := d.Decode(&actions); err != nil {
		if err == io.EOF {
			slog.Error("yaml actions file is empty")
			return nil, FailedParsingActionYamlEmpty
		} else {
			slog.Error("failed decoding yaml actions file", "err", err)
			return nil, FailedParsingActionYaml
		}
	}

	// enabled required on non-pointer fields
	v := validator.New(validator.WithRequiredStructEnabled())

	err := v.Struct(actions)
	if err != nil {
		slog.Error("failed validation of actions struct", "err", err)
		// NOTE: ideally we could return a printable error with the invalid fields so users could fix them
		return nil, FailedParsingActionYamlInvalid
	}

	if actions.Meta.CliVersion != common.CliVersion {
		slog.Warn("actions file created by different cli", "cli-version", actions.Meta.CliVersion)
	}

	if actions.Meta.SchemaVersion != SchemaVersion {
		slog.Warn("actions file created by different schema version", "input-version", actions.Meta.SchemaVersion, "current", SchemaVersion)
		inputSchema, err := semver.NewVersion(actions.Meta.SchemaVersion)
		if err != nil {
			slog.Error("invalid semver version for schema in actions file", "version", actions.Meta.SchemaVersion)
			return nil, FailedParsingActionYamlInvalid
		}

		currentSchema, _ := semver.NewVersion(SchemaVersion)

		if inputSchema.Major() != currentSchema.Major() {
			slog.Error("unsupported major version for schema in actions file", "version", actions.Meta.SchemaVersion)
			return nil, common.NewPrintableError("unsupported schema version - please use version %s", SchemaVersion)
		}
	}

	return actions, nil
}

func SaveActionFile(actionFile *ActionsFile, w io.Writer) error {
	yamlEncoder := yaml.NewEncoder(w)
	yamlEncoder.SetIndent(2)
	err := yamlEncoder.Encode(actionFile)
	if err != nil {
		return common.NewPrintableError("failed writing to actions file")
	}

	return nil
}
