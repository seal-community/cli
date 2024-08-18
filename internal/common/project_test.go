package common

import (
	"strings"
	"testing"
)

func TestNormalizeProjectName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"tag", "tag"},
		{"TAG", "TAG"},
		{" tag ", "-tag-"},
		{" TAG ", "-TAG-"},
		{"!#$tag%$@", "tag"},
		{"tag.-_tag", "tag.-_tag"},
		{"com.github.seal-sec/demo", "com.github.seal-sec-demo"},
		{strings.Repeat("A", 300), strings.Repeat("A", MaxProjectNameLen)},
		{"a/b/c\\d e", "a-b-c-d-e"},
		{"åß∂ƒ®å∑ƒ®tag", "tag"},
		{"😀tag😀", "tag"},
		{"github.com/Masterminds/goutils", "github.com-Masterminds-goutils"},
		{"golang.org/x/crypto", "golang.org-x-crypto"},
		{"@group/name", "group-name"},
		{"org:java:some:artifact", "orgjavasomeartifact"},
		{"python_module", "python_module"},
		{"python-module", "python-module"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := NormalizeProjectName(test.name)
			if result != test.expected {
				t.Errorf("expected %s, got %s", test.expected, result)
			}
		})
	}
}
