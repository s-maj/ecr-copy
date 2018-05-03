package main

import (
	"ecr_copy/cmd"
	log "github.com/sirupsen/logrus"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "15:04:05.000",
		FullTimestamp:   true,
	})
}

func main() {
	cmd.Execute()
}
