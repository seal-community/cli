//go:build windows

package common

func GetFileStats(path string) (*UnixStat, error) {
	return nil, nil
}
