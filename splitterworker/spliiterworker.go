package spliiterworker

import (
	"time"
	"vconvd/lib"
	"vconvd/logger"
	"vconvd/model"

	nsq "github.com/nsqio/go-nsq"
	"github.com/vmihailenco/msgpack"
)

var (
	log = logger.Log
)

type Config struct {
	NsqdHost  string
	NsqdPort  int
	NsqdTopic string
}

type SplitterWorker struct {
	Config   *Config
	Consumer *lib.NsqConsumer
	done     chan bool
}

type messageHandler struct{}

func (w *SplitterWorker) Start() {
	w.done = make(chan bool)
	w.Consumer = &lib.NsqConsumer{Host: w.Config.NsqdHost, Port: w.Config.NsqdPort, Topic: w.Config.NsqdTopic, Log: true}

	err := w.Consumer.Setup()
	if err != nil {
		log.Fatalf("Can not setup nsqd consumer: %s", err)
	}

	w.Consumer.Nsqc.AddHandler(&messageHandler{})

	err = w.Consumer.Connect()
	if err != nil {
		log.Fatalf("Can not connect to nsqd at %s:%d %s", w.Config.NsqdHost, w.Config.NsqdPort, err)
	} else {
		log.Debugf("Succesfully connected to nsqd: %s:%d", w.Config.NsqdHost, w.Config.NsqdPort)
	}

	<-w.done
}

func (w *SplitterWorker) Stop() {
	w.done <- true
}

func (h *messageHandler) HandleMessage(m *nsq.Message) error {
	var task model.ConversionTask
	err := msgpack.Unmarshal(m.Body, &task)
	if err != nil {
		log.Errorf("Can not unmarshal a message: %s", err)
		return err
	}

	log.Debugf("Got a message: %v", task)

	time.Sleep(time.Second * 10)
	return nil
}
