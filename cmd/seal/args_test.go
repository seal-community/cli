package main

import (
	"cli/internal/common"
	"testing"

	"github.com/spf13/cobra"
)

func TestExtractTarget(t *testing.T) {
	type input struct {
		args       []string
		filesystem string
		isOs       bool
	}
	type result struct {
		target string
		tt     common.TargetType
	}
	cases := []struct {
		input    input
		expected result
	}{
		{
			input: input{
				args:       []string{},
				filesystem: "",
				isOs:       false,
			},
			expected: result{
				target: "",
				tt:     common.ManifestTarget,
			},
		},
		{
			input: input{
				args:       []string{"/path/to/requirements.txt"},
				filesystem: "",
				isOs:       false,
			},
			expected: result{
				target: "/path/to/requirements.txt",
				tt:     common.ManifestTarget,
			},
		},
		{
			// Deprecated, should be removed in the future
			input: input{
				args:       []string{"/path/to/directory"},
				filesystem: "",
				isOs:       false,
			},
			expected: result{
				target: "/path/to/directory",
				tt:     common.ManifestTarget,
			},
		},
		{
			input: input{
				args:       []string{"/path/to/directory"},
				filesystem: "java",
				isOs:       false,
			},
			expected: result{
				target: "/path/to/directory",
				tt:     common.JavaFilesTarget,
			},
		},
		{
			input: input{
				args:       []string{},
				filesystem: "java",
				isOs:       false,
			},
			expected: result{
				target: "",
				tt:     common.JavaFilesTarget,
			},
		},
		{
			input: input{
				args:       []string{"os"},
				filesystem: "",
				isOs:       false,
			},
			expected: result{
				target: "",
				tt:     common.OsTarget,
			},
		},
		{
			input: input{
				args:       []string{},
				filesystem: "",
				isOs:       true,
			},
			expected: result{
				target: "",
				tt:     common.OsTarget,
			},
		},
	}

	for _, c := range cases {
		if target, tt := extractTarget(c.input.args, c.input.filesystem, c.input.isOs); target != c.expected.target || tt != c.expected.tt {
			t.Fatalf("got %v %v", target, tt)
		}
	}

}

func TestExtractArgArray(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringArray("test", []string{}, "")
	err := cmd.Flags().Set("test", "test0")
	if err != nil {
		t.Fatalf("failed to set flag")
	}
	if result := getArgArray(cmd, "test"); len(result) != 1 || result[0] != "test0" {
		t.Fatalf("got %v", result)
	}
}
