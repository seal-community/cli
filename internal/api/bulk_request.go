//go:build !mockserver
// +build !mockserver

package api

import "cli/internal/common"

type BulkCheckRequest struct {
	Entries  []common.Dependency    `json:"entries"`
	Metadata map[string]interface{} `json:"metadata"`
}
