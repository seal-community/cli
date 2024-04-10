package common

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

func Colorize(s string, c AnsiColor) string {
	isTty := term.IsTerminal(int(os.Stdout.Fd()))
	if !isTty {
		return s
	}
	
	return fmt.Sprintf("%s%s%s", c, s, AnsiReset)
}
