package worker

import (
	"fmt"
	"vconvd/logger"

	nsq "github.com/bitly/go-nsq"
)

var log = logger.Log

type NsqConsumer struct {
	Host  string
	Port  int
	Topic string
	Nsqc  *nsq.Consumer
	Log   bool
}

func (c *NsqConsumer) Setup() error {
	cfg := nsq.NewConfig()

	var err error
	c.Nsqc, err = nsq.NewConsumer(c.Topic, "put", cfg)
	if err != nil {
		log.Panic(err)
	}

	return c.Nsqc.ConnectToNSQD(fmt.Sprintf("%s:%d", c.Host, c.Port))
}
