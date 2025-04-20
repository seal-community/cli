package common

import "time"

var CliStartTime time.Time
var injectedStartTime string = ""

const StartTimeLayout string = "2006-01-02T15:04:05.000Z"

// in order to test the outputs of the cli, we need to be able to make the output constant
func init() {
	var err error
	if injectedStartTime == "" {
		CliStartTime = time.Now().UTC()
	} else {
		CliStartTime, err = time.Parse(StartTimeLayout, injectedStartTime)
		if err != nil {
			panic(err)
		}
	}
}
