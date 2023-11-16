package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/turbot/steampipe-plugin-sdk/v5/logging"
)

func setupLogger(plugin string) {
	level := logging.LogLevel()
	hcLevel := hclog.LevelFromString(level)

	options := &hclog.LoggerOptions{
		// make the name unique so that logs from this instance can be filtered
		Name:       fmt.Sprintf("[%s]", plugin),
		Level:      hcLevel,
		Output:     os.Stderr,
		TimeFn:     func() time.Time { return time.Now().UTC() },
		TimeFormat: "2006-01-02 15:04:05.000 UTC",
	}
	logger := logging.NewLogger(options)
	log.SetOutput(logger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true}))
	log.SetPrefix("")
	log.SetFlags(0)
}
