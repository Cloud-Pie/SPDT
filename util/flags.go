package util

import "flag"

type FlagVars struct {
	LogFile	bool
	ConfigFile string
	PricesFile string
	Port	string
}

func ParseFlags() FlagVars{
	logFile := flag.Bool("log-file",false,"Enable writing logs in file")
	configFile := flag.String("config-file", "config.yml", "Configuration file path")
	priceFile := flag.String("prices-file","", "Prices file path")
	port := flag.String("http-port","8083", "Http Port")
	flag.Parse()

	return FlagVars{
		 *logFile,
		 *configFile,
		 *priceFile,
		 *port,
	}
}
