package utils

import (
	"io"
	"os"
	"strings"
)

func ReadFromBIO(reader io.Reader) (string, error) {
	buf := new(strings.Builder)
	_, err := io.Copy(buf, reader)
	if err != nil {
		return "", err
	}

	return buf.String(), err
}

func CheckDirectoryExist(path string) {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(path, os.ModePerm)
	}
}
