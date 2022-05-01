package conversionworker

import (
	"time"
	"vconvd/lib"
	"vconvd/logger"
	"vconvd/model"

	"github.com/google/uuid"
	nsq "github.com/nsqio/go-nsq"
	"github.com/vmihailenco/msgpack"
)

var (
	log = logger.Log
)

type Config struct {
	NsqdHost         string
	NsqdPort         int
	NsqdManagerTopic string
	NsqdTopic        string
}

type ConversionWorker struct {
	Config           *Config
	producer         *lib.NsqProducer
	consumer         *lib.NsqConsumer
	worker           *model.Worker
	keepAliveStarted bool
	done             chan bool
}

func (w *ConversionWorker) Register() {
	w.done = make(chan bool)

	w.consumer = &lib.NsqConsumer{Host: w.Config.NsqdHost, Port: w.Config.NsqdPort, Topic: w.Config.NsqdTopic, Log: true}
	err := w.consumer.Setup()
	if err != nil {
		log.Fatalf("Can not setup nsqd consumer: %s", err)
	}

	w.consumer.Nsqc.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		w.HandleMessage(message)
		return nil
	}))

	err = w.consumer.Connect()
	if err != nil {
		log.Fatalf("Can not connect producer to nsqd at %s:%d %s", w.Config.NsqdHost, w.Config.NsqdPort, err)
	} else {
		log.Debugf("Producer succesfully connected to nsqd: %s:%d", w.Config.NsqdHost, w.Config.NsqdPort)
	}

	w.producer = &lib.NsqProducer{Host: w.Config.NsqdHost, Port: w.Config.NsqdPort, Log: true}
	err = w.producer.Setup()
	if err != nil {
		log.Fatalf("Can not connect consumer to nsqd at %s:%d", w.Config.NsqdHost, w.Config.NsqdPort)
	} else {
		log.Debugf("Consumer succesfully connected to nsqd: %s:%d", w.Config.NsqdHost, w.Config.NsqdPort)
	}

	worker := model.Worker{}
	worker.ID = string(uuid.New().String())
	w.worker = &worker

	task := model.Task{Name: "conversion-worker:register", Data: worker}
	data, err := msgpack.Marshal(task)
	if err != nil {
		log.Errorf("Failed to marshal a worker data to the msgpack format: %s", err)
		return
	}

	w.producer.Nsqp.Publish(w.Config.NsqdManagerTopic, data)

	<-w.done
}

func (w *ConversionWorker) KeepAlive() {
	if !w.keepAliveStarted {
		go func() {
			for true {
				task := model.Task{Name: "conversion-worker:ping", Data: w.worker}
				data, err := msgpack.Marshal(task)
				if err != nil {
					log.Errorf("Failed to marshal a task data to the msgpack format: %s", err)
					return
				}

				err = w.producer.Nsqp.Publish(w.Config.NsqdManagerTopic, data)
				if err != nil {
					log.Fatalf("Failed to publish the task to the queue %s:", err)
				}

				time.Sleep(time.Second * 5)
			}
		}()

		w.keepAliveStarted = true
	}
}

func (w *ConversionWorker) Stop() {
	w.done <- true
}

func (w *ConversionWorker) HandleMessage(message *nsq.Message) error {
	var task model.Task
	err := msgpack.Unmarshal(message.Body, &task)
	if err != nil {
		log.Errorf("Can not unmarshal a message: %s", err)
		message.Finish()
		return err
	}

	switch task.Name {
	case "conversion-worker:registered":
		log.Infof("Registered succesfully")
		w.KeepAlive()
	}

	message.Finish()
	return nil
}
