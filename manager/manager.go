package manager

import (
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/nsqio/go-nsq"
	"github.com/vmihailenco/msgpack"

	"vconvd/lib"
	"vconvd/logger"
	"vconvd/model"
)

var log = logger.Log

type Config struct {
	NsqdHost            string
	NsqdPort            int
	NsqdManagerTopic    string
	NsqdSplitterTopic   string
	NsqdConversionTopic string
	NsqdJoinerTopic     string
	RestHost            string
	RestPort            int
	DbFile              string
}

type Manager struct {
	Config      *Config
	producer    *lib.NsqProducer
	consumer    *lib.NsqConsumer
	rest        *Rest
	dataStorage *DataStorage
	convworkers map[string]*model.Worker

	doneChan chan bool
}

func New(config *Config) *Manager {
	m := Manager{Config: config}

	return &m
}

func (m *Manager) Stop() {
	m.doneChan <- true
}

func (m *Manager) Run() {
	m.dataStorage = &DataStorage{DbFile: m.Config.DbFile}
	m.convworkers = make(map[string]*model.Worker)
	m.doneChan = make(chan bool)

	m.ensureDatabase()
	defer m.dataStorage.Close()

	m.producer = &lib.NsqProducer{
		Host: m.Config.NsqdHost,
		Port: m.Config.NsqdPort,
		Log:  true,
	}
	err := m.producer.Setup()
	if err != nil {
		log.Fatalf("Can not connect the producer to nsqd at %s:%d", m.Config.NsqdHost, m.Config.NsqdPort)
	} else {
		log.Debugf("Producer succesfully connected to nsqd: %s:%d", m.Config.NsqdHost, m.Config.NsqdPort)
	}
	defer m.producer.Stop()

	m.consumer = &lib.NsqConsumer{
		Host:  m.Config.NsqdHost,
		Port:  m.Config.NsqdPort,
		Topic: m.Config.NsqdManagerTopic,
		Log:   true,
	}
	err = m.consumer.Setup()
	if err != nil {
		log.Fatalf("Can not setup nsqd consumer: %s", err)
	}

	m.consumer.Nsqc.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		m.handleMessage(message)
		return nil
	}))

	err = m.consumer.Connect()
	if err != nil {
		log.Fatalf("Can not connect the consumer to nsqd at %s:%d %s", m.Config.NsqdHost, m.Config.NsqdPort, err)
	} else {
		log.Debugf("Consumer succesfully connected to nsqd: %s:%d", m.Config.NsqdHost, m.Config.NsqdPort)
	}

	go m.convWorkersGC()

	m.rest = &Rest{manager: m, config: &RestConfig{
		RestHost: m.Config.RestHost,
		RestPort: m.Config.RestPort,
	}}

	go m.rest.Run()

	<-m.doneChan
	m.rest.StopAndWait()
}

func (m *Manager) ensureDatabase() {
	if _, err := os.Stat(m.Config.DbFile); errors.Is(err, os.ErrNotExist) {
		log.Debugf("Creating database %s", m.Config.DbFile)
		err = m.dataStorage.CreateNewDb()
		if err != nil {
			log.Fatalf("Can not create database %s: %s", m.Config.DbFile, err)
		}
	}

	err := m.dataStorage.Open()
	if err != nil {
		log.Fatalf("Can not open database %s: %s", m.Config.DbFile, err)
	}
}

func (m *Manager) handleMessage(message *nsq.Message) error {
	var task model.Task
	err := msgpack.Unmarshal(message.Body, &task)
	if err != nil {
		log.Errorf("Can not unmarshal a message: %s", err)
		message.Finish()
		return err
	}
	task.Message = message

	log.Debugf("Got the task: %s", task.Name)

	switch task.Name {
	case "conversion-worker:register":
		m.registerConvWorkerTask(&task)
	case "conversion-worker:ping":
		m.pingConvWorkerTask(&task)
	case "conversion:put":
		m.createTaskTask(&task)
	}

	return nil
}

func (m *Manager) registerConvWorkerTask(task *model.Task) {
	defer task.Message.Finish()

	var worker model.Worker
	//TODO: it is not safe. replace this code with someting else
	mapstructure.Decode(task.Data, &worker)

	log.Infof("Registering worker %s", worker.ID)

	m.convworkers[worker.ID] = &worker

	rTask := model.Task{Name: "conversion-worker:registered", Data: worker}

	data, err := msgpack.Marshal(rTask)
	if err != nil {
		log.Errorf("Failed to marshal a task data to the msgpack format: %s", err)
		return
	}
	err = m.producer.Nsqp.Publish(m.Config.NsqdConversionTopic, data)
	if err != nil {
		log.Errorf("Failed to publish task: %s", err)
		return
	}
}

func (m *Manager) pingConvWorkerTask(task *model.Task) {
	var worker model.Worker
	mapstructure.Decode(task.Data, &worker)
	if _, ok := m.convworkers[worker.ID]; !ok {
		m.registerConvWorkerTask(task)
	}

	m.convworkers[worker.ID].LastPing = time.Now()
	task.Message.Finish()
}

func (m *Manager) createTaskTask(task *model.Task) {
	if len(m.convworkers) == 0 {
		log.Errorf("There is no active workers. Requeue after 5 min.")
		task.Message.RequeueWithoutBackoff(time.Second * 5)
		return
	}

	var convtask model.ConversionTask
	mapstructure.Decode(task.Data, &convtask)
	err := m.CreateTask(&convtask)
	if err != nil {
		log.Errorf("Failed to create the task: %s", err)
	}

	task.Message.Finish()
}

func (m *Manager) CreateTask(convtask *model.ConversionTask) error {
	convtask.ID = uuid.New().String()
	cworkersCount := len(m.convworkers)

	if cworkersCount == 0 {
		return m.taskQueue(convtask, time.Second*5)
	}

	chunksLen, err := m.getChunksLength(convtask)
	if err != nil {
		m.taskQueue(convtask, time.Minute*10)
		return fmt.Errorf("Can not probe video file: %s", err)
	}
	if chunksLen == 0 {
		m.taskQueue(convtask, time.Minute*10)
		return fmt.Errorf("Got zero chunks length for some reason")
	}

	chunks := m.getChunks(cworkersCount, chunksLen)
	convtask.Chunks = chunks

	err = m.dataStorage.CreateTask(convtask)
	if err != nil {
		m.taskQueue(convtask, time.Minute*10)
		return fmt.Errorf("Failed to create task in the database: %s", err)
	}

	for _, chunk := range convtask.Chunks {
		splitTask := model.SplitTask{InputFile: convtask.InputFile, Chunk: chunk}
		err = m.chunkQueue(&splitTask)
		if err != nil {
			//TODO: remove the task completly
			return fmt.Errorf("Can not queue a chunk: %s - removing the task", err)
		}
	}

	return nil
}

func (m *Manager) removeTask(convtask *model.ConversionTask) {

}

func (m *Manager) getChunksLength(convtask *model.ConversionTask) (float64, error) {
	ffmpegh := lib.FFMpegHelper{}
	err := ffmpegh.Parse(convtask.InputFile)
	if err != nil {
		return 0, err
	}

	videolen, err := ffmpegh.GetLength()
	if err != nil {
		return 0, err
	}

	chunklen := math.Ceil(videolen / float64(len(m.convworkers)))

	return chunklen, nil
}

func (m *Manager) getChunks(chunksCount int, chunksLen float64) []*model.Chunk {
	var chunks []*model.Chunk

	for i := 0; i < chunksCount; i++ {
		var chunk *model.Chunk = new(model.Chunk)
		chunk.Sequence = uint32(i) + 1
		chunk.Offset = float64(i) * chunksLen
		chunk.Length = chunksLen - 1
		chunk.Status = model.ChunkPendingStatus
		chunks = append(chunks, chunk)
	}

	return chunks
}

func (m *Manager) taskQueue(convtask *model.ConversionTask, delay time.Duration) error {
	task := model.Task{Name: "conversion:put", Data: convtask}
	data, err := msgpack.Marshal(task)
	if err != nil {
		log.Errorf("Failed to marshal a task data to the msgpack format: %s", err)
		return fmt.Errorf("Failed to marshal a task data to the msgpack format: %s", err)
	}

	err = m.producer.Nsqp.DeferredPublish(m.Config.NsqdManagerTopic, delay, data)
	if err != nil {
		log.Errorf("Failed to publish the task %s to the queue: %s", convtask.ID, err)
	} else {
		log.Debugf("Pushed to nsqd a new task: %s", convtask.ID)
	}

	return fmt.Errorf("There is no active workers - delayed")
}

func (m *Manager) chunkQueue(chunk *model.SplitTask) error {
	task := model.Task{Name: "conversion:split", Data: chunk}
	data, err := msgpack.Marshal(task)
	if err != nil {
		return err
	}

	err = m.producer.Nsqp.Publish(m.Config.NsqdSplitterTopic, data)
	if err != nil {
		return err
	}

	return nil
}

func (m *Manager) convWorkersGC() {
	for true {
		now := time.Now()
		for _, worker := range m.convworkers {
			diff := now.Sub(worker.LastPing)
			if diff.Seconds() > 10 {
				log.Debugf("Unregistering worker %s due to last ping time", worker.ID)
				delete(m.convworkers, worker.ID)
			}
		}

		time.Sleep(time.Second * 5)
	}
}
