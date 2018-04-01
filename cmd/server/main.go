package main

import (
	"flag"

	log "github.com/cihub/seelog"
	"proxy/config"
	"proxy/server"
)

func main() {
	config_file := flag.String("config", "./config/config.toml", "Input your server configure file")
	log_file := flag.String("logconfig", "./config/logcfg.xml", "Input your log configure file")

	flag.Parse()

	logger, err := log.LoggerFromConfigAsFile(*log_file)
	if err != nil {
		log.Critical("err parsing config log file", err)
		return
	}
	log.ReplaceLogger(logger)
	defer log.Flush()

	serverCfg, err := config.NewServerConfWithFile(*config_file)
	if err != nil {
		log.Error(err)
		return
	}

	log.Debug(serverCfg)

	Service, err := server.NewService(serverCfg)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("service start")
	Service.Run()

}
