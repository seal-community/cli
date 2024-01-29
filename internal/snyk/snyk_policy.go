package snyk

import (
	"bytes"
	"cli/internal/common"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type ruleNodeMap struct {
	node  *yaml.Node            // holds the node for the issue's value, so new rules could be added
	rules map[string]*yaml.Node // value doesn't matter, just for convenience checks
}

type issueMap map[string]ruleNodeMap
type PolicyFile struct {
	root       yaml.Node
	ignore     *yaml.Node      // to inject new issue id
	issues     issueMap        // to inject new rule path object
	createTime time.Time       // used for all new rules added
	newRules   map[string]bool // used to prevent addition of rules with duplicate values
}

// references for uses in the wild https://github.com/search?q=path%3A%22%2F.snyk%22&type=code
const PolicyFileName = ".snyk"

const fixReason = "Fixed by Seal Security"

// template used from https://docs.snyk.io/scan-using-snyk/policies/the-.snyk-file
const snykFileSchemaVersion = "v1.25.0"

var snykTemplateHeader string = fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: %s
`, snykFileSchemaVersion) // if contained `ignore: {}`` it caused the other values to use `inline` formatting as well isntead of `flow``

func validateSnykFile(r io.Reader) error {
	snykFile := struct {
		Version string `yaml:"version"`
	}{}

	if err := yaml.NewDecoder(r).Decode(&snykFile); err != nil {
		slog.Error("failed parsing snyk file", "err", err)
		return common.WrapWithPrintable(err, "failed to parse .snyk file")
	}

	if snykFile.Version != snykFileSchemaVersion {
		slog.Error("mismatched snyk file schema version", "version", snykFile.Version)
		return common.NewPrintableError("unsupported .snyk file schema %s", snykFile.Version)
	}

	return nil
}

func formatRule(pkg string, version string) string {
	return fmt.Sprintf("* > %s@%s", pkg, version)
}

func formatCreatedTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05.000Z")
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

// this is used to find all the nodes for existing issues, rules and save them in a convenient structure for querying
func loadSnykIssueMap(ignore *yaml.Node) (issueMap, error) {
	issues := make(issueMap)
	// find all existing issues and their rules
	currentIssueId := ""
	for i, issueIdNode := range ignore.Content {
		if i%2 == 0 { // stored as key,value in array
			// grab the issue id (the key in the map) and skip to the actual value node
			currentIssueId = issueIdNode.Value
			continue
		}

		slog.Debug("loading issue rules", "issue", currentIssueId)
		if _, exists := issues[currentIssueId]; exists {
			// this can affect the map, since our rule could be a duplicate of the first instance, but not the second;
			// for example, if we want to add `lodash@4.15.17` to `SNYK_123` we wouldn't detect it for the following existing yaml:
			//		SNYK_123:
			//			"lodash@4.15.17"
			//				...
			//		SNYK_123:
			//			"semver-regex@1.0.0"
			//				...
			slog.Warn("duplicate issue found", "issue", currentIssueId)
			// we are merging the lists so we don't lose any rule paths, this does not matter much since its used to check for dups, and won't affect the actual structure of yaml output
			// would cause us to create our new rule in the first intance of the issue
		} else {
			issues[currentIssueId] = ruleNodeMap{node: issueIdNode, rules: make(map[string]*yaml.Node)}
		}

		for _, rulePathMapNode := range issueIdNode.Content {
			// the rule paths are saved in a sequence, which contains a map
			if len(rulePathMapNode.Content) != 2 {
				// assuming each entry is a map that only contains 1 value - the rule path as key ("* > lodash"), and the rule object as value (with reason/created values)
				slog.Error("not enough child nodes in issue - skipping", "count", len(rulePathMapNode.Content))
				return nil, common.NewPrintableError("unsupported .snyk file schema - too many child nodes in line:%d column:%d", rulePathMapNode.Line, rulePathMapNode.Column)
			}

			ruleNode := rulePathMapNode.Content[0]
			if ruleNode.Kind != yaml.ScalarNode {
				// the rule node should be a scalar, a string representing the filter
				slog.Error("wrong kind for rule path - skipping", "kind", ruleNode.Kind)
				return nil, common.NewPrintableError("unsupported .snyk file schema - wrong child nodes in line:%d column:%d", rulePathMapNode.Line, rulePathMapNode.Column)
			}

			rulePath := ruleNode.Value
			payload := rulePathMapNode.Content[1]
			slog.Debug("loading rule", "issue", currentIssueId, "rule", rulePath)

			// don't care about duplicate here, since it's just used to prevent us from editing the yaml and will overwrite the old value
			// using the latest one to hopefully add our rule at the bottom
			issues[currentIssueId].rules[rulePath] = payload
		}
	}

	return issues, nil
}

func buildIgnoreRule(ruleFilter string, createTime time.Time) *yaml.Node {
	// this creates the array of maps, containing the rule metadata as leaf
	// 		- '* > lodash@4.15.17':
	// 			reason: Fixed by Seal Security
	// 			created: %s
	return &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: ruleFilter,
				Style: yaml.SingleQuotedStyle, // to make this always use single quotes; testable
			},
			{
				Kind: yaml.MappingNode, // maps are stored as arrays of: key,value,key,value...
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "reason"},
					{Kind: yaml.ScalarNode, Value: fixReason},
					{Kind: yaml.ScalarNode, Value: "created"},
					{Kind: yaml.ScalarNode, Value: formatCreatedTime(createTime)},
				},
			},
		},
	}
}

func _buildInternalRuleId(issueId string, pkg string, version string) string {
	return fmt.Sprintf("%s|%s|%s", issueId, pkg, version)
}

// these path components are actually in the dependency graph see https://docs.snyk.io/snyk-cli/commands/ignore#path-less-than-path_to_resource-greater-than
// e.g. `chokidar > fsevents > node-pre-gyp > tar-pack`
//
// issue ID is usually `SNYK_...` but can be something else like `npm:angular:123123`
func (pf *PolicyFile) AddRule(issueId string, pkg string, version string) bool {
	ruleFilter := formatRule(pkg, version)
	slog.Info("adding ignore rule for snyk yaml", "issue", issueId, "rule", ruleFilter)
	ruleid := _buildInternalRuleId(issueId, pkg, version)
	if _, exists := pf.newRules[ruleid]; exists {
		// must not add duplicate issues, causes snyk cli / scm integration to fail
		// seems like a dup rulepath still works, but this should prevent them as well
		slog.Warn("rule was already added", "id", ruleid)
		return false
	}

	ignoreRuleNode := buildIgnoreRule(ruleFilter, pf.createTime)

	added := false
	issue, exists := pf.issues[issueId]
	if !exists {
		// needs to create the map of issue-id -> array of rules
		// the key is the issue id, the value is an array containing the a map of our new rule
		key := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: issueId,
		}

		value := &yaml.Node{
			Kind:    yaml.SequenceNode,
			Style:   yaml.DoubleQuotedStyle,
			Content: []*yaml.Node{ignoreRuleNode},
		}

		slog.Info("adding new issue with rule", "issue", issueId, "rule", ruleFilter)
		pf.ignore.Content = append(pf.ignore.Content, key, value)
		added = true
	} else {
		// issue id already exists
		if _, exists := issue.rules[ruleFilter]; !exists {
			slog.Info("adding rule to existing issue", "issue", issueId, "rule", ruleFilter)
			issue.node.Content = append(issue.node.Content, ignoreRuleNode)
			added = true

		} else {
			slog.Info("skipped rule since it already exists", "issue", issueId, "rule", ruleFilter)
		}
	}

	if added {
		pf.newRules[ruleid] = true // don't care about the value
	}

	return added
}

func addIgnoreNode(root *yaml.Node) *yaml.Node {
	key := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "ignore",
	}

	ignoreNodeValue := &yaml.Node{
		Kind:    yaml.MappingNode,
		Content: []*yaml.Node{},
	}

	root.Content[0].Content = append(root.Content[0].Content, key, ignoreNodeValue)
	return ignoreNodeValue
}

func decodeSnykFile(r io.Reader) (*PolicyFile, error) {
	var root yaml.Node

	err := yaml.NewDecoder(r).Decode(&root)
	if err != nil { // unlikely
		slog.Error("failed decoding yaml", "err", err)
		return nil, common.WrapWithPrintable(err, "failed to decode .snyk file")
	}

	ignore := findIgnoreNode(&root)
	if ignore == nil {
		slog.Warn("ignore section not found")
		ignore = addIgnoreNode(&root)
	} else {
		if ignore.Kind != yaml.MappingNode {
			slog.Error("bad ignore node kind", "kind", ignore.Kind, "line", ignore.Line, "column", ignore.Column)
			return nil, common.NewPrintableError(".snyk file parsing error in line:%d column:%d", ignore.Line, ignore.Column)
		}

		if ignore.Content == nil {
			// init in case it did not exist, should not happen
			slog.Warn("creating new array for empty ignore node")
			ignore.Content = make([]*yaml.Node, 0)
		}
	}

	issues, err := loadSnykIssueMap(ignore)
	if err != nil {
		return nil, err
	}

	pf := PolicyFile{
		root:       root,
		issues:     issues,
		ignore:     ignore,
		newRules:   make(map[string]bool),
		createTime: time.Now().UTC(), // same as outputted by snyk when generated from commandline
	}

	return &pf, nil
}

// using raw Node unmarshal/marshal to minimize changes done to the data; which would happen if we loaded and saved to a struct
func LoadPolicy(r io.Reader) (*PolicyFile, error) {

	// copying input from reader so we can validate and also parse the yaml into a tree
	inputCopy := bytes.NewBufferString("")
	tee := io.TeeReader(r, inputCopy)

	if err := validateSnykFile(tee); err != nil {
		return nil, err // already printable
	}

	return decodeSnykFile(inputCopy)
}

func SavePolicy(pf *PolicyFile, w io.Writer) error {
	doc := pf.root.Content[0]
	e := yaml.NewEncoder(w)

	e.SetIndent(2) // seems to be the one used by snyk
	return e.Encode(doc)
}

func NewPolicy() (*PolicyFile, error) {
	// loading from template since it's the easiest way
	pf, err := LoadPolicy(strings.NewReader(snykTemplateHeader))
	if err != nil {
		slog.Error("failed loading template snyk policy document", "err", err)
		return nil, common.WrapWithPrintable(err, "failed to generate new .snyk file")
	}
	return pf, nil
}
