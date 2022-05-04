package main

import (
	"os"
	"vconvd/logger"
	"vconvd/splitterworker"

	"github.com/urfave/cli"
)

var log = logger.Log

func main() {
	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "nsqd-host",
			Value: "127.0.0.1",
			Usage: "nsqd host",
		},
		cli.StringFlag{
			Name:  "nsqd-port",
			Value: "4150",
			Usage: "nsqd port",
		},
		cli.StringFlag{
			Name:  "nsqd-topic",
			Value: "vconvd-splitter",
			Usage: "nsqd topic",
		},
		cli.StringFlag{
			Name:  "chunk-path",
			Value: "/tmp",
			Usage: "chunk temp path",
		},
		cli.StringFlag{
			Name:  "log-file",
			Usage: "log to given file",
		},
		cli.BoolFlag{
			Name:  "log-stderr-disable",
			Usage: "disable log to stderr",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "verbose logging",
		},
	}

	app.Name = "vconvd-spliter-worker"
	app.Version = "1.0.0"
	app.Usage = "videoconvd splitter worker"
	app.Before = func(c *cli.Context) error {
		var logLevel string
		if c.Bool("verbose") {
			logLevel = "DEBUG"
		} else {
			logLevel = "INFO"
		}

		logger.SetupLogger(logger.Config{LogFile: c.String("log-file"), LogLevel: logLevel})
		if !c.Bool("log-stderr-disable") {
			cli.ShowVersion(c)
		}

		return nil
	}
	app.Action = func(c *cli.Context) error {
		log.Infof("Starting splitter worker")

		config := &splitterworker.Config{
			NsqdHost:  c.String("nsqd-host"),
			NsqdPort:  c.Int("nsqd-port"),
			NsqdTopic: c.String("nsqd-topic"),
			ChunkPath: c.String("chunk-path"),
		}
		w := splitterworker.SplitterWorker{Config: config}
		w.Start()
		return nil
	}

	app.Run(os.Args)
}
