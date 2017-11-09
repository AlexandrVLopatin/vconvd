package model

type ConversionTask struct {
	ID            string
	ProducerID    string                      `json:"producer_id"`
	InputFile     string                      `json:"input_file"`
	OutputFile    string                      `json:"output_file"`
	FFMpegArgs    string                      `json:"ffmpeg_args"`
	Thumbnails    []ConversionTaskThumbnail   `json:"thumbnails"`
	HTTPCallbacks ConversionTaskHTTPCallbacks `json:"callbacks"`
}

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
