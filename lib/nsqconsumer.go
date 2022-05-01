package lib

import (
	"fmt"

	nsq "github.com/nsqio/go-nsq"
)

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
	return err
}

func (c *NsqConsumer) Connect() error {
	return c.Nsqc.ConnectToNSQD(fmt.Sprintf("%s:%d", c.Host, c.Port))
}
