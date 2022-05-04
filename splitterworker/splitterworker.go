package splitterworker

import (
	"fmt"
	"path/filepath"
	"vconvd/lib"
	"vconvd/logger"
	"vconvd/model"

	"github.com/mitchellh/mapstructure"
	nsq "github.com/nsqio/go-nsq"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
	"github.com/vmihailenco/msgpack"
)

var log = logger.Log

type Config struct {
	NsqdHost         string
	NsqdPort         int
	NsqdManagerTopic string
	NsqdTopic        string
	ChunkPath        string
}

type SplitterWorker struct {
	Config   *Config
	consumer *lib.NsqConsumer
	producer *lib.NsqProducer
	done     chan bool
}

type messageHandler struct{}

func (w *SplitterWorker) Start() {
	w.done = make(chan bool)

	w.consumer = &lib.NsqConsumer{
		Host:  w.Config.NsqdHost,
		Port:  w.Config.NsqdPort,
		Topic: w.Config.NsqdTopic,
		Log:   true,
	}

	err := w.consumer.Setup()
	if err != nil {
		log.Fatalf("Can not setup nsqd consumer: %s", err)
	}

	w.consumer.Nsqc.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		w.handleMessage(message)
		return nil
	}))

	err = w.consumer.Connect()
	if err != nil {
		log.Fatalf("Can not connect consumer to nsqd at %s:%d %s", w.Config.NsqdHost, w.Config.NsqdPort, err)
	} else {
		log.Debugf("Succesfully connected to nsqd: %s:%d", w.Config.NsqdHost, w.Config.NsqdPort)
	}

	w.producer = &lib.NsqProducer{
		Host: w.Config.NsqdHost,
		Port: w.Config.NsqdPort,
		Log:  true,
	}
	err = w.producer.Setup()
	if err != nil {
		log.Fatalf("Can not connect producer to nsqd at %s:%d %s", w.Config.NsqdHost, w.Config.NsqdPort, err)
	} else {
		log.Debugf("Producer succesfully connected to nsqd: %s:%d", w.Config.NsqdHost, w.Config.NsqdPort)
	}

	<-w.done
}

func (w *SplitterWorker) Stop() {
	w.done <- true
}

func (w *SplitterWorker) handleMessage(m *nsq.Message) error {
	var task model.Task
	err := msgpack.Unmarshal(m.Body, &task)
	if err != nil {
		log.Errorf("Can not unmarshal a message: %s", err)
		return err
	}

	log.Debugf("Got a message: %v", task)

	err = nil
	switch task.Name {
	case "conversion:split":
		err = w.split(&task)
	}

	if err != nil {
		log.Errorf("%s", err)
	}

	return nil
}

func (w *SplitterWorker) split(task *model.Task) error {
	var splitTask model.SplitTask
	mapstructure.Decode(task.Data, &splitTask)

	path := filepath.FromSlash(fmt.Sprintf("%s/%s_%d%s",
		w.Config.ChunkPath,
		splitTask.ID,
		splitTask.Chunk.Sequence,
		filepath.Ext(splitTask.InputFile),
	))

	st := model.Task{Name: "splitter-worker:start", Data: model.SplitStartedTask{ID: splitTask.ID, ChunkFile: path}}
	data, err := msgpack.Marshal(st)
	if err != nil {
		return fmt.Errorf("Failed to marshal a SplitStartedTask to the msgpack format: %s", err)
	}
	err = w.producer.Nsqp.Publish(w.Config.NsqdManagerTopic, data)
	if err != nil {
		return fmt.Errorf("Failed to pubslish a SplitStartedTask to the queue: %s", err)
	}

	err = ffmpeg_go.
		Input(splitTask.InputFile, ffmpeg_go.KwArgs{
			"ss": splitTask.Chunk.Offset,
			"t":  splitTask.Chunk.Length,
		}).
		Output(path, ffmpeg_go.KwArgs{
			"vcodec": "copy",
			"acodec": "copy",
		}).
		ErrorToStdOut().
		Run()

	if err != nil {
		return fmt.Errorf("Splitting error: %s", err)
	}

	ft := model.Task{Name: "splitter-worker:finish", Data: model.SplitFinishedTask{ID: splitTask.ID}}
	data, err = msgpack.Marshal(ft)
	if err != nil {
		return fmt.Errorf("Failed to marshal a SplitFinishTask to the msgpack format: %s", err)
	}
	err = w.producer.Nsqp.Publish(w.Config.NsqdManagerTopic, data)
	if err != nil {
		return fmt.Errorf("Failed to pubslish a SplitStartedTask to the queue: %s", err)
	}

	return err
}
