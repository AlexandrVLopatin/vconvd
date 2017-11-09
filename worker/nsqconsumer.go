package worker

import nsq "github.com/bitly/go-nsq"

type NsqComsumer struct {
	Host string
	Port string
	Nsqc *nsq.Consumer
	Log  bool
}
