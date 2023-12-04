package common

import (
	"errors"
	"fmt"
)

var errorPrefix string = Colorize("Error", AnsiNiceRed)

func AsPrintable(err error) *PrintableError {
	var e *PrintableError
	errors.As(err, &e)
	return e
}

type PrintableError struct {
	msg string
	err error // wrapped
}

func (e *PrintableError) Unwrap() error {
	return e.err
}

func (e *PrintableError) Error() string {
	return fmt.Sprintf("%s: %s", errorPrefix, e.msg)
}

func NewPrintableError(s string, a ...any) *PrintableError {
	return WrapWithPrintable(nil, s, a...)
}

func WrapWithPrintable(err error, s string, a ...any) *PrintableError {
	msg := fmt.Sprintf(s, a...)
	return &PrintableError{msg: msg, err: err}
}

// returns joined errors where the original error is higher priority, in case it is printable
// will allow using errors.As to fetch the most inner printable-error thanks to using depth-first traversal (https://pkg.go.dev/errors#As)
func FallbackPrintableMsg(err error, s string, a ...any) error {
	return errors.Join(err, NewPrintableError(s, a...))
}
