package common

import (
	"log/slog"
	"net/url"
	"path/filepath"
)

// return the last component of url, empty string on error
func FileNameFromUri(u string) string {
	parsed, err := url.Parse(u)
	if err != nil {
		slog.Debug("failed parsing uri", "u", u)
		return ""
	}

	return FileNameFromURL(parsed)
}

func FileNameFromURL(u *url.URL) string {
	if u == nil {
		slog.Debug("url was nil")
		return ""
	}

	filename := filepath.Base(filepath.FromSlash(u.Path))
	return filename
}
