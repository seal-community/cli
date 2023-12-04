package phase

import (
	"fmt"
	"testing"
)

func TestProjectNameValid(t *testing.T) {

	for _, projName := range []string{
		"test",
		"my_proj",
		"my-proj",
		"my-proj1",
		"my.proj1",
		".",
		"1",
		"a1",
		"aA1",
		"a",
		"a..",
		"1.",
		".1.",
	} {
		t.Run(fmt.Sprintf("name_%s", projName), func(t *testing.T) {
			if msg := validateProjectName(projName); msg != "" {
				t.Fatalf("incorrectly checked valid project name `%s` : `%s`", projName, msg)
			}
		})
	}
}

func TestProjectNameInvalid(t *testing.T) {

	for _, projName := range []string{
		"a ",
		" a",
		"my proj",
		"a,",
	} {
		t.Run(fmt.Sprintf("name_%s", projName), func(t *testing.T) {
			if msg := validateProjectName(projName); msg == "" {
				t.Fatalf("project name should be invalid `%s` : `%s`", projName, msg)
			}
		})
	}
}
