package snyk

import (
	"bytes"
	"cli/internal/common"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRuleFormat(t *testing.T) {
	result := formatRule("my-pckg", "1.2.30")
	if result != "* > my-pckg@1.2.30" {
		t.Fatalf("got %s", result)
	}
}

func TestCreatedTimeFormat(t *testing.T) {
	ct := time.Unix(1, 814000000).UTC()
	result := formatCreatedTime(ct)
	if result != "1970-01-01T00:00:01.814Z" {
		t.Fatalf("got %s", result)
	}
}

func TestSnykPolicySanityNew(t *testing.T) {

	pf, err := NewPolicy()
	if err != nil || pf == nil {
		t.Fatalf("failed creating empty %v", err)
	}

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}

	result := w.String()
	expected := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore: {}
`
	if result != expected {
		t.Fatalf("did empty policy bad creation:\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}
func TestSnykPolicySanityNewWithIgnoreRule(t *testing.T) {

	pf, err := NewPolicy()
	if err != nil || pf == nil {
		t.Fatalf("failed creating empty %v", err)
	}

	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-FOLLOWREDIRECTS-6141137", "follow-redirects", "1.15.2-sp1")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: %s
`, formatCreatedTime(ct))
	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}
func TestSnykPolicySanityMarshalled(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}

	result := w.String()
	if original != result {
		t.Fatalf("did not marshall policy the same way it was read:\nOriginal:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", original, result)
	}
}

func TestSnykPolicySanityNoIgnoreAddEmptyIgnore(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore: {}
`
	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityEmptyIgnoreWithSpace(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:  # has extra space
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err == nil || pf != nil {
		t.Fatalf("should have error for bad ignore pf:%v err:%v", pf, err)
	}
	if printableErr := common.AsPrintable(err); printableErr == nil {
		t.Fatalf("should build a printable error")
	}
}

func TestSnykPolicySanityEmptyIgnoreNewLine(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err == nil || pf != nil {
		t.Fatalf("should have error for bad ignore pf:%v err:%v", pf, err)
	}
	if printableErr := common.AsPrintable(err); printableErr == nil {
		t.Fatalf("should build a printable error")
	}
}

func TestSnykPolicySanityNoIgnoreAddIssue(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-FOLLOWREDIRECTS-6141137", "follow-redirects", "1.15.2-sp1")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: %s
`, formatCreatedTime(ct))
	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityAddToExistingIssue(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-FOLLOWREDIRECTS-6141137", "lodash", "4.15.17")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z
    - '* > lodash@4.15.17':
        reason: Fixed by Seal Security
        created: %s
`, formatCreatedTime(ct))
	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityAddToExistingIssueAndOtherFields(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z
patch:
  'npm:request:20160119':
    - inbound > tldextract > request:
        patched: '2019-04-16T08:41:31.200Z'
  SNYK-JS-LODASH-450202:
    - '@google-cloud/storage > async > lodash':
        patched: '2019-07-05T23:03:26.426Z'
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-FOLLOWREDIRECTS-6141137", "lodash", "4.15.17")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z
    - '* > lodash@4.15.17':
        reason: Fixed by Seal Security
        created: %s
patch:
  'npm:request:20160119':
    - inbound > tldextract > request:
        patched: '2019-04-16T08:41:31.200Z'
  SNYK-JS-LODASH-450202:
    - '@google-cloud/storage > async > lodash':
        patched: '2019-07-05T23:03:26.426Z'
`, formatCreatedTime(ct))
	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityDoesNotAddToExistingRule(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: No Reason
        created: 2024-01-25T19:31:14.000Z
    - '* > lodash@4.15.17':
        reason: No reason
        created: 2024-01-25T19:31:14.000Z
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-FOLLOWREDIRECTS-6141137", "lodash", "4.15.17")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()

	if original != result {
		t.Fatalf("did not marshall policy the same way it was read:\nOriginal:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", original, result)
	}
}

func TestSnykPolicySanityAddNewIssue(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-123123-123123", "lodash", "4.15.17")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z
  SNYK-JS-123123-123123:
    - '* > lodash@4.15.17':
        reason: Fixed by Seal Security
        created: %s
`, formatCreatedTime(ct))
	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityAddNewIssueNonAnsi(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-Cç§:
    - '* > follow-redirects@1.15.2-sp1':
        reason: これはぺんです
        created: 2024-01-25T19:31:14.000Z
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-123123-123123", "lodash", "4.15.17")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-Cç§:
    - '* > follow-redirects@1.15.2-sp1':
        reason: これはぺんです
        created: 2024-01-25T19:31:14.000Z
  SNYK-JS-123123-123123:
    - '* > lodash@4.15.17':
        reason: Fixed by Seal Security
        created: %s
`, formatCreatedTime(ct))
	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityAddNewIssueDupRule(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc
        created: 2024-01-25T19:31:14.000Z
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc123123
        created: 2024-01-25T19:31:14.000Z
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-123123-123123", "lodash", "4.15.17")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc
        created: 2024-01-25T19:31:14.000Z
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc123123
        created: 2024-01-25T19:31:14.000Z
  SNYK-JS-123123-123123:
    - '* > lodash@4.15.17':
        reason: Fixed by Seal Security
        created: %s
`, formatCreatedTime(ct))

	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityAddNewIssueDupIssue(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc
        created: 2024-01-25T19:31:14.000Z
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc123123
        created: 2024-01-25T19:31:14.000Z
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-123123-123123", "lodash", "4.15.17")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc
        created: 2024-01-25T19:31:14.000Z
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc123123
        created: 2024-01-25T19:31:14.000Z
  SNYK-JS-123123-123123:
    - '* > lodash@4.15.17':
        reason: Fixed by Seal Security
        created: %s
`, formatCreatedTime(ct))

	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityAddNewIssueDupIssueMerged(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc
        created: 2024-01-25T19:31:14.000Z
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > lodash@4.15.17':
        reason: abc123123
        created: 2024-01-25T19:31:14.000Z
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-FOLLOWREDIRECTS-6141137", "semver-regex", "1.0.0")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > follow-redirects@1.15.2-sp1':
        reason: abc
        created: 2024-01-25T19:31:14.000Z
    - '* > semver-regex@1.0.0':
        reason: Fixed by Seal Security
        created: %s
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    - '* > lodash@4.15.17':
        reason: abc123123
        created: 2024-01-25T19:31:14.000Z
`, formatCreatedTime(ct))

	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}

func TestSnykPolicySanityKeepComments(t *testing.T) {
	original := `# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0 # a comment
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    # comment
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z # another
`
	pf, err := LoadPolicy(strings.NewReader(original))
	if err != nil || pf == nil {
		t.Fatalf("failed loading %v", err)
	}
	ct := time.Now().UTC()
	pf.createTime = ct
	pf.AddRule("SNYK-JS-123123-123123", "lodash", "4.15.17")

	w := bytes.NewBufferString("")
	err = SavePolicy(pf, w)
	if err != nil {
		t.Fatalf("failed dumping ")
	}
	result := w.String()
	expected := fmt.Sprintf(`# Snyk (https://snyk.io) policy file, patches or ignores known vulnerabilities.
version: v1.25.0 # a comment
ignore:
  SNYK-JS-FOLLOWREDIRECTS-6141137:
    # comment
    - '* > follow-redirects@1.15.2-sp1':
        reason: Fixed by Seal Security
        created: 2024-01-25T19:31:14.000Z # another
  SNYK-JS-123123-123123:
    - '* > lodash@4.15.17':
        reason: Fixed by Seal Security
        created: %s
`, formatCreatedTime(ct))
	if result != expected {
		t.Fatalf("\nexpected:\n`%s`\n~~~~~~~~~~~\nGenerated:\n`%s`\n", expected, result)
	}
}
