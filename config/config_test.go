package config

import (
	"testing"
	"github.com/Cloud-Pie/SPDT/internal/util"
)

func TestFileFormat(t *testing.T) {
	_,err := ParseConfigFile("Config.yml")
	if err != nil {
		t.Error(
			"For", util.CONFIG_FILE,
			"expected", nil,
			"got", err,
		)
	}
}
