package restserver

import "github.com/bitly/go-nsq"

type NsqpProducer struct {
	Host string
	Port string
	Nsqp *nsq.Producer
	Log  bool
}

func (p *NsqpProducer) Setup() error {
	nsqconfig := nsq.NewConfig()
	p.Nsqp, _ = nsq.NewProducer(p.Host+":"+p.Port, nsqconfig)
	if !p.Log {
		p.Nsqp.SetLogger(nil, 0)
	}
	return p.Nsqp.Ping()
}

func (p *NsqpProducer) Stop() {
	p.Nsqp.Stop()
}
