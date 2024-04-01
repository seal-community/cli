package common

import (
	"encoding/json"
	"io"
	"log/slog"
	"os"

	"github.com/iancoleman/orderedmap"
)

func JsonLoad(path string) *orderedmap.OrderedMap {
	o := orderedmap.New()
	o.SetEscapeHTML(false) // required to be set before decoding, so that all nested ordered maps have it set as well

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("failed opening json file", "err", err, "path", path)
		return nil
	}

	if err := json.Unmarshal(data, &o); err != nil {
		slog.Error("failed loading json", "err", err, "path", path)
		return nil
	}

	return o
}

func JsonSave(projectAssets *orderedmap.OrderedMap, path string) error {
	w, err := CreateFile(path)
	if err != nil {
		slog.Error("failed opening json file", "err", err, "path", path)
		return err
	}
	defer w.Close()

	return JsonDump(projectAssets, w)
}

func JsonDump(projectAssets *orderedmap.OrderedMap, w io.Writer) error {
	e := json.NewEncoder(w)
	e.SetIndent("", "  ")  // 2 spaces
	e.SetEscapeHTML(false) // also required for output, otherwise '<=' would get escaped; setting it on the OrderedMap struct did not work

	if err := e.Encode(projectAssets); err != nil {
		slog.Error("failed saving json", "err", err)
		return err
	}

	return nil
}
