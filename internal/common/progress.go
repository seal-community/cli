package common

import (
	"github.com/schollz/progressbar/v3"
)

func NewProgressBar(visible bool, maxSteps int) *progressbar.ProgressBar {
	bar := progressbar.NewOptions(maxSteps, // can be changed with bar.ChangeMax()
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetVisibility(visible),
		progressbar.OptionSetWidth(15),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowDescriptionAtLineEnd(),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionClearOnFinish(),
		// progressbar.OptionShowElapsedTimeOnFinish(), //mutex with OptionClearOnFinish
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]|[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))

	return bar
}

func ScopedProgressBar(visible bool, maxSteps int, f func(*progressbar.ProgressBar) error) error {
	bar := NewProgressBar(visible, maxSteps)
	defer bar.Finish()
	return f(bar)
}
