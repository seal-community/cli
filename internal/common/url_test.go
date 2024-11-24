package common

import (
	"net/url"
	"testing"
)

func TestFileNameFromUri(t *testing.T) {
	if fn := FileNameFromUri("/qweqwe/baubau.bin"); fn != "baubau.bin" {
		t.Fatalf("wrong file name `%s`", fn)
	}

	if fn := FileNameFromUri("qweqwe/baubau.bin"); fn != "baubau.bin" {
		t.Fatalf("wrong file name `%s`", fn)
	}

	if fn := FileNameFromUri("https://example.com/qweqwe/baubau.bin"); fn != "baubau.bin" {
		t.Fatalf("wrong file name `%s`", fn)
	}
}

func TestFileNameFromURL(t *testing.T) {
	if fn := FileNameFromURL(nil); fn != "" {
		t.Fatalf("wrong file name `%s`", fn)
	}

	u, err := url.Parse("/qweqwe/baubau.bin")
	if err != nil {
		t.Fatalf("err %v", err)
	}

	if fn := FileNameFromURL(u); fn != "baubau.bin" {
		t.Fatalf("wrong file name `%s`", fn)
	}
}
