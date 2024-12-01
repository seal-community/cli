package main

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestExtractTarget(t *testing.T) {
	if result := extractTarget([]string{"param0"}); result != "param0" {
		t.Fatalf("got %s", result)
	}
}

func TestExtractTargetMultiple(t *testing.T) {
	if result := extractTarget([]string{"param0", "param1", "param2"}); result != "param0" {
		t.Fatalf("got %s", result)
	}
}

func TestExtractTargetEmpty(t *testing.T) {
	if result := extractTarget([]string{}); result != "" {
		t.Fatalf("got %s", result)
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
