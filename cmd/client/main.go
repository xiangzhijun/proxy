package main

import (
	"flag"

	log "github.com/cihub/seelog"
	"proxy/client"
	"proxy/config"
)

func main() {
	config_file := flag.String("config", "./config/config.toml", "Input your configure file")
	log_file := flag.String("logconfig", "./config/logcfg.xml", "Input your log configure file")

	flag.Parse()

	logger, err := log.LoggerFromConfigAsFile(*log_file)
	if err != nil {
		log.Critical("err parsing config log file", err)
		return
	}
	log.ReplaceLogger(logger)
	defer log.Flush()

	clientCfg, err := config.NewClientConfWithFile(*config_file)
	if err != nil {
		log.Error(err)
		return
	}

	Client := client.NewClient(clientCfg)

	Client.Run()
}
