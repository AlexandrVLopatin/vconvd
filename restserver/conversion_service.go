package restserver

import (
	"fmt"
	"net/http"
	"time"
	"vconvd/logger"
	"vconvd/model"

	"github.com/emicklei/go-restful"
	"github.com/vmihailenco/msgpack"
)

var log = logger.Log

type ConversionServiceConfig struct {
	HTTPHost  string
	HTTPPort  int
	NsqdHost  string
	NsqdPort  int
	NsqdTopic string
}

type ConversionService struct {
	config   *ConversionServiceConfig
	producer *NsqProducer

	blockChan chan bool
	doneChan  chan bool
}

func New(config *ConversionServiceConfig) *ConversionService {
	c := ConversionService{config: config}
	c.blockChan, c.doneChan = make(chan bool), make(chan bool)
	return &c
}

func (c *ConversionService) Run() {
	c.producer = &NsqProducer{Host: c.config.NsqdHost, Port: c.config.NsqdPort, Log: true}

	err := c.producer.Setup()
	if err != nil {
		log.Fatalf("Can not connect to nsqd at %s:%d", c.config.NsqdHost, c.config.NsqdPort)
	} else {
		log.Debugf("Succesfully connected to nsqd: %s:%d", c.config.NsqdHost, c.config.NsqdPort)
	}

	c.register()

	go func() {
		http.ListenAndServe(fmt.Sprintf("%s:%d", c.config.HTTPHost, c.config.HTTPPort), nil)
	}()

	<-c.blockChan

	log.Debug("Shutting down conversion service")
	c.producer.Stop()
	time.Sleep(time.Second)

	c.doneChan <- true
}

func (c *ConversionService) Stop() {
	close(c.blockChan)
}

func (c *ConversionService) WaitForFinish() {
	<-c.doneChan
}

func (c *ConversionService) StopAndWait() {
	c.Stop()
	c.WaitForFinish()
}

func (c ConversionService) register() {
	ws := new(restful.WebService)
	ws.
		Path("/").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.PUT("").To(c.putTask))
	ws.Route(ws.GET("/{id}").To(c.getTaskInfo))

	restful.Add(ws)
}

func (c *ConversionService) putTask(req *restful.Request, resp *restful.Response) {
	task := model.ConversionTask{}
	err := req.ReadEntity(&task)
	if err != nil {
		log.Errorf("Failed to read request entity: %s", err)
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}

	log.Debugf("Put a new task: %s", task.ID)

	data, err := msgpack.Marshal(task)
	if err != nil {
		log.Errorf("Failed to marshal a task data to the msgpack format: %s", err)
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}

	err = c.producer.Nsqp.Publish("vconvd", data)
	if err != nil {
		log.Errorf("Failed to publish the task %s to the queue: %s", task.ID, err)
		resp.WriteError(http.StatusInternalServerError, err)
	} else {
		log.Debugf("Pushed to nsqd a new task: %s", task.ID)
		resp.WriteHeaderAndEntity(http.StatusCreated, task)
	}
}

func (c *ConversionService) getTaskInfo(req *restful.Request, resp *restful.Response) {
	id := req.PathParameter("id")
	log.Debugf("Get task info: %s", id)
	resp.WriteHeaderAndEntity(http.StatusOK, id)
}
