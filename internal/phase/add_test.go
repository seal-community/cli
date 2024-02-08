package phase

import (
	"cli/internal/actions"
	"testing"
)

func TestAddRuleSanity(t *testing.T) {

	r := AddRule{
		From: actions.Override{Library: "ejs", Version: "2.7.4"},
		To:   &actions.Override{Library: "ejs", Version: "2.7.4-sp1"},
	}

	if r.isLatest() {
		t.Fatalf("should not be latest")
	}
	if r.isSafest() {
		t.Fatalf("should not be safest")
	}
}

func TestAddRuleLatestEmpty(t *testing.T) {

	r := AddRule{
		From: actions.Override{Library: "ejs", Version: "2.7.4"},
		To:   &actions.Override{}, // empty values
	}

	if !r.isLatest() {
		t.Fatalf("should be latest")
	}

	if r.isSafest() {
		t.Fatalf("not safest")
	}
}

func TestAddRuleLatestEmptyVersion(t *testing.T) {

	r := AddRule{
		From: actions.Override{Library: "ejs", Version: "2.7.4"},
		To:   &actions.Override{Library: "ejs"}, // empty values
	}

	if !r.isLatest() {
		t.Fatalf("should be latest")
	}

	if r.isSafest() {
		t.Fatalf("should not be safest")
	}
}

func TestAddRuleLatestEmptyLibrary(t *testing.T) {

	r := AddRule{
		From: actions.Override{Library: "ejs", Version: "2.7.4"},
		To:   &actions.Override{Version: "2.7.4-sp1"}, // shouldn't be valid input
	}

	if !r.isLatest() {
		t.Fatalf("should be latest")
	}

	if r.isSafest() {
		t.Fatalf("shouldnot be safest")
	}
}

func TestAddRuleSafest(t *testing.T) {

	r := AddRule{
		From: actions.Override{Library: "ejs", Version: "2.7.4"},
		To:   nil,
	}

	if r.isLatest() {
		t.Fatalf("should not be latest")
	}

	if !r.isSafest() {
		t.Fatalf("not safest")
	}
}
