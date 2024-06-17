package shared

import "cli/internal/common"

var ManagerProcessFailed = common.NewPrintableError("failed running package manager")
var FailedParsingManagerOutput = common.NewPrintableError("failed parsing package manager output")
