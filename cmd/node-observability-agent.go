package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sherine-k/node-observability-agent/pkg/server"
	log "github.com/sirupsen/logrus"
)

var (
	version       = "unknown"
	app           = "node-observability-agent"
	port          = flag.Int("port", 9000, "server port to listen on (default: 9000)")
	storageFolder = flag.String("storage", "/tmp/pprofs/", "folder to which the pprof files are saved")
	tokenFile     = flag.String("tokenFile", "", "file containing token to be used for kubelet profiling http request")
	logLevel      = flag.String("loglevel", "info", "log level")
	versionFlag   = flag.Bool("v", false, "print version")
	appVersion    = fmt.Sprintf("%s %s", app, version)
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println(appVersion)
		os.Exit(0)
	}

	lvl, err := log.ParseLevel(*logLevel)
	if err != nil {
		log.Errorf("Log level %s not recognized, using info", *logLevel)
		*logLevel = "info"
		lvl = log.InfoLevel
	}
	log.SetLevel(lvl)
	log.Infof("Starting %s at log level %s", appVersion, *logLevel)

	server.Start(server.Config{
		Port:          *port,
		TokenFile:     *tokenFile,
		StorageFolder: *storageFolder,
	})
}
