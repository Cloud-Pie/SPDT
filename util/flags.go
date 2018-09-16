package util

import "flag"

type FlagVars struct {
	LogFile	bool
}

func ParseFlags() FlagVars{
	logFile := flag.Bool("log-file",false,"Enable writing logs in file")
	flag.Parse()

	return FlagVars{
		 *logFile,
	}
}
