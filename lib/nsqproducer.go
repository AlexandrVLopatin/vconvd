package lib

import (
	"fmt"

	"github.com/nsqio/go-nsq"
)

type NsqProducer struct {
	Host string
	Port int
	Nsqp *nsq.Producer
	Log  bool
}

func (p *NsqProducer) Setup() error {
	cfg := nsq.NewConfig()
	p.Nsqp, _ = nsq.NewProducer(fmt.Sprintf("%s:%d", p.Host, p.Port), cfg)
	if !p.Log {
		p.Nsqp.SetLogger(nil, 0)
	}
	return p.Nsqp.Ping()
}

func (p *NsqProducer) Stop() {
	p.Nsqp.Stop()
}
