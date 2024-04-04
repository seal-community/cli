package main

import (
	"testing"
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
