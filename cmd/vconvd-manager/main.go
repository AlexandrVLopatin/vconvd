package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/urfave/cli"

	"vconvd/logger"
	"vconvd/manager"
)

var (
	log = logger.Log
	m   *manager.Manager
)

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
			Name:  "nsqd-manager-topic",
			Value: "vconvd-manager",
			Usage: "nsqd manager topic",
		},
		cli.StringFlag{
			Name:  "nsqd-conversion-topic",
			Value: "vconvd-conversion",
			Usage: "nsqd topic",
		},
		cli.StringFlag{
			Name:  "rest-host",
			Value: "127.0.0.1",
			Usage: "REST host",
		},
		cli.IntFlag{
			Name:  "rest-port",
			Value: 8089,
			Usage: "REST port",
		},
		cli.StringFlag{
			Name:  "log-file",
			Usage: "log to given file",
		},
		cli.StringFlag{
			Name:  "db-file",
			Value: "vconvd.bd",
			Usage: "database file path",
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

	app.Name = "vconvd-manager"
	app.Version = "1.0.0"
	app.Usage = "videoconvd manager"
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
		log.Infof("Starting Manager")
		setupSigHandlers()

		config := &manager.Config{
			NsqdHost:            c.String("nsqd-host"),
			NsqdPort:            c.Int("nsqd-port"),
			NsqdManagerTopic:    c.String("nsqd-manager-topic"),
			NsqdConversionTopic: c.String("nsqd-conversion-topic"),
			RestHost:            c.String("rest-host"),
			RestPort:            c.Int("rest-port"),
			DbFile:              c.String("db-file"),
		}
		m = &manager.Manager{Config: config}
		m.Run()

		log.Info("Gracefully stopped")
		return nil
	}

	app.Run(os.Args)
}

func setupSigHandlers() {
	signalch := make(chan os.Signal, 1)

	signal.Notify(
		signalch,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGABRT,
	)

	go func() {
		sig := <-signalch

		log.Warningf("Received an %s signal.", sig)
		m.Stop()
	}()
}
