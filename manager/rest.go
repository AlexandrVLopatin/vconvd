package manager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	R "github.com/go-pkgz/rest"

	"vconvd/model"
)

type RestConfig struct {
	RestHost string
	RestPort int
}

type Rest struct {
	config  *RestConfig
	manager *Manager

	blockChan chan bool
	doneChan  chan bool
}

func (c *Rest) Run() {
	log.Infof("Starting REST server at %s:%d", c.config.RestHost, c.config.RestPort)
	c.blockChan, c.doneChan = make(chan bool), make(chan bool)

	router := c.getRouter()

	go func() {
		http.ListenAndServe(fmt.Sprintf("%s:%d", c.config.RestHost, c.config.RestPort), router)
	}()

	<-c.blockChan
	time.Sleep(time.Second)
	c.doneChan <- true
}

func (c *Rest) Stop() {
	log.Debug("Stopping rest server")
	close(c.blockChan)
}

func (c *Rest) WaitForFinish() {
	<-c.doneChan
}

func (c *Rest) StopAndWait() {
	c.Stop()
	c.WaitForFinish()
}

func (c *Rest) getRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.Timeout(10 * time.Second))

	r.Put("/", c.putTaskAction)
	r.Get("/{id}", c.getTaskInfoAction)

	return r
}

func (c *Rest) putTaskAction(w http.ResponseWriter, r *http.Request) {
	convTask := model.ConversionTask{}
	err := json.NewDecoder(r.Body).Decode(&convTask)
	if err != nil {
		log.Errorf("Failed to read request entity: %s", err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	log.Debugf("Put a new task: %s", convTask.ID)

	err = c.manager.CreateConvTask(&convTask)
	if err != nil {
		http.Error(w, string(err.Error()), 400)
	}
	render.JSON(w, r, convTask)
}

func (c *Rest) getTaskInfoAction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	log.Debugf("Get task info: %s", id)

	render.JSON(w, r, R.JSON{"id": id})
}
