package composer

import (
	"cli/internal/config"
	"fmt"
	"testing"
)

type Pair struct {
	First  string
	Second string
}

func TestNormalizePackageNames(t *testing.T) {
	c, _ := config.New(nil)
	manager := NewComposerManager(c, "./composer.lock", "")
	names := []Pair{
		{"aaaaa", "aaaaa"},
		{"aaAAa", "aaaaa"},
		{"AAAAA", "aaaaa"},
		{"AAa_a", "aaa_a"},
		{"aAa/Aa", "aaa/aa"},
	}
	for i, pair := range names {
		t.Run(fmt.Sprintf("name_%d", i), func(t *testing.T) {
			if manager.NormalizePackageName(pair.First) != pair.Second {
				t.Fatalf("failed to normalize `%s`", pair)
			}
		})
	}
}

func TestNormalizePackageVersion(t *testing.T) {
	versions := []Pair{
		{"1.2.3", "1.2.3"},
		{"v1.2.3", "1.2.3"},
	}
	for i, pair := range versions {
		t.Run(fmt.Sprintf("version_%d", i), func(t *testing.T) {
			if normalizePackageVersion(pair.First) != pair.Second {
				t.Fatalf("failed to normalize `%s`", pair)
			}
		})
	}
}

type InOutPair struct {
	Input    string
	Expected bool
}

func TestIsVersionSupported(t *testing.T) {
	c, _ := config.New(nil)
	manager := NewComposerManager(c, "./composer.lock", "")
	expected := []InOutPair{
		{"1.2.3", false},
		{"2.0.0", true},
		{"2.0.1", true},
	}

	for i, pair := range expected {
		t.Run(fmt.Sprintf("version_%d", i), func(t *testing.T) {
			if manager.IsVersionSupported(pair.Input) != pair.Expected {
				t.Fatalf("failed to check version `%s`", pair.Input)
			}
		})
	}
}

func TestGetProjectName(t *testing.T) {
	c, _ := config.New(nil)
	manager := NewComposerManager(c, "testdata/project/composer.lock", "testdata/project")
	expected := "test/test"
	if manager.GetProjectName() != expected {
		t.Fatalf("failed to get project name")
	}
}
