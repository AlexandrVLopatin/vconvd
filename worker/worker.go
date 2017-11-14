package worker

import (
	nsq "github.com/bitly/go-nsq"
)

type Config struct {
	NsqdHost  string
	NsqdPort  int
	NsqdTopic string
}

type Worker struct {
	Config   *Config
	Consumer *NsqConsumer
	done     chan bool
}

func (w *Worker) Start() {
	w.done = make(chan bool)
	w.Consumer = &NsqConsumer{Host: w.Config.NsqdHost, Port: w.Config.NsqdPort, Topic: w.Config.NsqdTopic, Log: true}
	w.Consumer.Setup()
	w.Consumer.Nsqc.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		log.Debugf("Got a message: %v", message)
		return nil
	}))
	<-w.done
}

func (w *Worker) Stop() {
	w.done <- true
}
