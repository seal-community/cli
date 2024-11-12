package api

import "testing"

func TestParamExists(t *testing.T) {
	if !paramExists([]StringPair{{Name: "param-name", Value: ""}}, "param-name") {
		t.Fatal("failed finding param name")
	}
}

func TestParamExistsNotFound(t *testing.T) {
	if paramExists([]StringPair{{Name: "param-name", Value: ""}}, "new-name") {
		t.Fatal("failed finding param name")
	}
}

func TestCalculateQuerystringLengthEmpty(t *testing.T) {
	if l := calculateQuerystringLength(nil); l != 0 {
		t.Fatalf("bad length fro no params: %d", l)
	}
}

func TestCalculateQuerystringLengthSanity(t *testing.T) {
	expected := len("param=myval")
	if l := calculateQuerystringLength([]StringPair{{Name: "param", Value: "myval"}}); l != expected {
		t.Fatalf("bad length for params: %d exepcted %d", l, expected)
	}
}
