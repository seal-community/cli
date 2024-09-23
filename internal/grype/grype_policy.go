package grype

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"io"
	"log/slog"
	"strings"

	"gopkg.in/yaml.v3"
)

const PolicyFileName = ".grype.yaml"

type PolicyFile struct {
	root          yaml.Node
	ignore        *yaml.Node      // to inject new rules
	existingVulns map[string]bool // used to prevent addition of rules with duplicate values
}

const fixReason = "Fixed by Seal Security"

// Based on https://github.com/anchore/syft/blob/92d63d/syft/pkg/type.go#L10
func grypePackageManager(pkgManager string) string {
	switch pkgManager {
	case mappings.PythonManager:
		return "python"
	case mappings.NpmManager:
		return "npm"
	case mappings.GolangManager:
		return "go-module"
	case mappings.MavenManger:
		return "java-archive"
	case mappings.NugetManager:
		return "dotnet"
	default:
		// pkgManager should be one of the supported package managers
		// if it is not, we return an empty string to avoid adding the type field
		// this should not happen, all package managers supported by CLI should be supported here
		return ""
	}
}

func grypePkgName(pkg string, pkgManager string) string {
	if pkgManager == mappings.MavenManger {
		// Maven packages are in the format `group:artifact`
		// we need to drop the group name
		//
		// Example:
		// com.fasterxml.jackson.core:jackson-databind -> jackson-databind
		// org.apache.commons:commons-lang3 -> commons-lang3
		parts := strings.Split(pkg, ":")
		return parts[max(0, len(parts)-1)]
	}
	return pkg
}

// Creates a new ignore rule for the given package and version
// https://github.com/anchore/grype/tree/v0.80.0?tab=readme-ov-file#specifying-matches-to-ignore
func buildIgnoreRule(vulnId, pkg, version, pkgManager string) *yaml.Node {
	grypePkgManager := grypePackageManager(pkgManager)

	// add the package information for ignoring
	// we don't add `location` since the CLI fixes everywhere
	packageContent := []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "name"},
		{Kind: yaml.ScalarNode, Value: grypePkgName(pkg, pkgManager)},
		{Kind: yaml.ScalarNode, Value: "version"},
		{Kind: yaml.ScalarNode, Value: version},
	}

	if grypePkgManager == "" {
		slog.Warn("unknown package manager", "manager", pkgManager)
	} else {
		packageContent = append(packageContent,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "type"},
			&yaml.Node{Kind: yaml.ScalarNode, Value: grypePkgManager},
		)
	}

	return &yaml.Node{
		// maps are stored as arrays of: key,value,key,value...
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "vulnerability",
				Style: yaml.TaggedStyle, // Use yaml.TaggedStyle to avoid quoting since that's how grype does it
			},
			{
				Kind:  yaml.ScalarNode,
				Value: vulnId,
				Style: yaml.TaggedStyle,
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "reason",
				Style: yaml.TaggedStyle,
			},
			{
				Kind:  yaml.ScalarNode,
				Value: fixReason,
				Style: yaml.TaggedStyle,
			},
			{
				Kind:  yaml.ScalarNode,
				Value: "package",
				Style: yaml.TaggedStyle,
			},
			{
				Kind:    yaml.MappingNode,
				Content: packageContent,
			},
		},
	}
}

func (pf *PolicyFile) AddRule(vulnId string, pkg string, version string, pkgManager string) bool {
	if _, exists := pf.existingVulns[vulnId]; exists {
		slog.Warn("grype ignore rule already exists", "id", vulnId)
		return false
	}

	slog.Info("adding ignore rule for .grype.yaml", "vuln", vulnId)

	ignoreRuleNode := buildIgnoreRule(vulnId, pkg, version, pkgManager)

	pf.ignore.Content = append(pf.ignore.Content, ignoreRuleNode)
	pf.existingVulns[vulnId] = true
	return true
}

func addIgnoreNode(root *yaml.Node) *yaml.Node {
	key := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "ignore",
	}

	ignoreNodeValue := &yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: []*yaml.Node{},
	}

	root.Content[0].Content = append(root.Content[0].Content, key, ignoreNodeValue)
	return ignoreNodeValue
}

func findIgnoreNode(root *yaml.Node) *yaml.Node {
	if len(root.Content) == 0 {
		return nil
	}

	found := false
	for _, child := range root.Content[0].Content {
		if child.Value == "ignore" && child.Kind == yaml.ScalarNode {
			found = true
			continue
		}

		if found {
			return child
		}
	}

	return nil
}

func LoadPolicy(r io.Reader) (*PolicyFile, error) {
	// See unit tests for examples of the expected format
	var root yaml.Node

	err := yaml.NewDecoder(r).Decode(&root)
	if err != nil { // unlikely
		slog.Error("failed decoding yaml", "err", err)
		return nil, common.WrapWithPrintable(err, "failed to decode .grype.yaml file")
	}

	ignore := findIgnoreNode(&root)
	if ignore == nil {
		slog.Warn("ignore section not found")
		ignore = addIgnoreNode(&root)
	} else {
		if ignore.Kind != yaml.SequenceNode {
			slog.Error("bad ignore node kind", "kind", ignore.Kind, "line", ignore.Line, "column", ignore.Column)
			return nil, common.NewPrintableError(".grype.yaml file parsing error in line:%d column:%d", ignore.Line, ignore.Column)
		}

		if ignore.Content == nil {
			// init in case it did not exist, should not happen
			slog.Warn("creating new array for empty ignore node")
			ignore.Content = make([]*yaml.Node, 0)
		}
	}

	existingVulns := make(map[string]bool)
	for _, ignoreEntry := range ignore.Content {
		for i, field := range ignoreEntry.Content {
			if field.Kind == yaml.ScalarNode && field.Value == "vulnerability" {
				if i+1 >= len(ignoreEntry.Content) {
					slog.Warn("bad ignore entry, missing vuln id", "entry", ignoreEntry)
					return nil, common.NewPrintableError(".grype.yaml file parsing error")
				}

				vulnId := ignoreEntry.Content[i+1].Value
				slog.Debug("adding existing ignore rule", "rule", vulnId)
				existingVulns[vulnId] = true
			}
		}
	}

	pf := PolicyFile{
		root:          root,
		ignore:        ignore,
		existingVulns: existingVulns,
	}

	return &pf, nil
}

func SavePolicy(pf *PolicyFile, w io.Writer) error {
	doc := pf.root.Content[0]
	e := yaml.NewEncoder(w)

	e.SetIndent(2) // seems to be the one used by grype
	return e.Encode(doc)
}

func NewPolicy() (*PolicyFile, error) {
	root := yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{
				Kind:    yaml.MappingNode,
				Content: make([]*yaml.Node, 0),
			},
		},
	}

	ignore := addIgnoreNode(&root)
	pf := &PolicyFile{
		root:          root,
		ignore:        ignore,
		existingVulns: make(map[string]bool),
	}
	return pf, nil
}
