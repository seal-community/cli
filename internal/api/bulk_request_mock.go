//go:build mockserver
// +build mockserver

package api

import "cli/internal/common"

// needs to ifdef out the metadata field from being sent due to environment changes affecting body

type BulkCheckRequest struct {
	Entries  []common.Dependency    `json:"entries"`
	Metadata map[string]interface{} `json:"-"`
}
