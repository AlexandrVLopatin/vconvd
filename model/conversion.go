package model

import (
	"time"

	"github.com/nsqio/go-nsq"
)

type Worker struct {
	ID       string
	LastPing time.Time
}

type TaskError struct {
	code    int
	message string
}

type Task struct {
	Message *nsq.Message
	Name    string      `json:"name"`
	Data    interface{} `json:"data"`
}

type ConversionTask struct {
	ID            string                       `json:"id"`
	ProducerID    string                       `json:"producer_id"`
	InputFile     string                       `json:"input_file"`
	OutputFile    string                       `json:"output_file"`
	FFMpegArgs    string                       `json:"ffmpeg_args"`
	Thumbnails    []*ConversionTaskThumbnail   `json:"thumbnails"`
	HTTPCallbacks *ConversionTaskHTTPCallbacks `json:"callbacks"`
	Chunks        []*Chunk                     `json:"chunks"`
}

type Chunk struct {
	Sequence uint32  `json:"sequence"`
	Offset   float64 `json:"offset"`
	Length   float64 `json:"length"`
	File     string  `json:"file"`
	Status   uint8   `json:"status"`
}

type SplitTask struct {
	InputFile string `json:"input_file"`
	Chunk     *Chunk `json:"chunk"`
}

const (
	ChunkPendingStatus = iota
	ChunkWorkingStatus = iota
)

type ConversionTaskThumbnail struct {
	Size       uint   `json:"size"`
	Quality    byte   `json:"quality"`
	Seek       string `json:"seek"`
	OutputFile string `json:"ouput_file"`
}

type ConversionTaskHTTPCallbacks struct {
	Before   string `json:"before"`
	After    string `json:"after"`
	Error    string `json:"error"`
	Progress string `json:"progress"`
}
