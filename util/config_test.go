package util

import (
	"testing"
)

func TestFileFormat(t *testing.T) {
	_,err := ReadConfigFile("config_test.yml")
	if err != nil {
		t.Error(
			"For: ", "config_test.yml",
			"expected: ", nil,
			"got: ", err,
		)
	}
}
