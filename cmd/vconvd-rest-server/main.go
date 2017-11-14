package main

import (
	"os"
	"os/signal"
	"syscall"

	"vconvd/logger"
	"vconvd/restserver"

	"github.com/urfave/cli"
)

var (
	log    = logger.Log
	server *restserver.ConversionService
)

func main() {
	app := cli.NewApp()

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "host",
			Value: "127.0.0.1",
			Usage: "service host",
		},
		cli.IntFlag{
			Name:  "port",
			Value: 8089,
			Usage: "service port",
		},
		cli.StringFlag{
			Name:  "nsqd-host",
			Value: "127.0.0.1",
			Usage: "nsqd host",
		},
		cli.IntFlag{
			Name:  "nsqd-port",
			Value: 4150,
			Usage: "nsqd port",
		},
		cli.StringFlag{
			Name:  "nsqd-topic",
			Value: "vconvd",
			Usage: "nsqd topic",
		},
		cli.StringFlag{
			Name:  "log-file",
			Usage: "log to given file",
		},
		cli.BoolFlag{
			Name:  "verbose",
			Usage: "verbose logging",
		},
	}

	app.Name = "vconvd-reset-server"
	app.Version = "1.0.0"
	app.Usage = "videoconvd REST API server"
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
		setupSigHandlers()

		log.Infof("Start listening on http://%s:%d", c.String("host"), c.Int("port"))

		server = restserver.New(&restserver.ConversionServiceConfig{
			HTTPHost:  c.String("host"),
			HTTPPort:  c.Int("port"),
			NsqdHost:  c.String("nsqd-host"),
			NsqdPort:  c.Int("nsqd-port"),
			NsqdTopic: c.String("nsqd-topic"),
		})

		server.Run()

		log.Info("Stopped")
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
		server.StopAndWait()
	}()
}
