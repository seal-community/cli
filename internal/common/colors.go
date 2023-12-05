package common

type AnsiColor string

// IMPORTANT: this colors will not work for windows legacy console
const (
	AnsiReset          = "\033[0m"
	AnsiFaint          = "\033[2m"
	AnsiBrightRed      = "\033[91m"
	AnsiBrightGreen    = "\033[92m"
	AnsiBrightYellow   = "\033[93m"
	AnsiBrightRedFaint = "\033[91;2m"
	AnsiNiceRed        = "\033[38;5;1m"
	AnsiLightGrey      = "\033[38;5;249m"
	AnsiOrange         = "\033[38;5;180m"
	AnsiLightRed       = "\033[38;5;174m"
	AnsiWarnYellow     = "\033[38;5;184m"
	AnsiDarkGrey       = "\033[90m"
	AnsiColdPurple     = "\033[95m"
	AnsiBlue           = "\033[36m"
)
