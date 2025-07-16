package main

import (
	"flag"

	log "github.com/sirupsen/logrus"

	"app/common"
	"app/render"
)

func main() {
	// Load configuration
	configFile := flag.String("config", "", "Path to configuration YAML file")
	flag.Parse()
	if *configFile == "" {
		log.Fatal("-config not given.")
	}
	config, err := common.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Fail to load config file from %v. %v", *configFile, err)
	}

	// Setup Logger
	common.SetupLogger(config)
	log.Debug(config)

	// Setup Render
	game := render.NewGame(config)
	render.RunGame(game)
}
