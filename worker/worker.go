package worker

type WorkerConfig {
    NsqdHost string
    NsqdPort string
}

type Worker struct {
    config *WorkerConfig
}

func Start() {
    config := nsq.NewConfig()
	q, _ := nsq.NewConsumer("vconvd", "ch", config)
	q.AddHandler(nsq.HandlerFunc(func(message *nsq.Message) error {
		log.Printf("Got a message: %v", message)
		wg.Done()
		return nil
	}))
	err := q.ConnectToNSQD("127.0.0.1:4150")
	if err != nil {
		log.Panic("Could not connect")
	}
}