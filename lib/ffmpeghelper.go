package lib

import (
	"strconv"

	"github.com/Jeffail/gabs/v2"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
)

type FFMpegHelper struct {
	Filepath string
	JSON     gabs.Container
}

func (f *FFMpegHelper) Parse(filepath string) error {
	f.Filepath = filepath

	probe, err := ffmpeg_go.Probe(filepath)
	if err != nil {
		return err
	}

	jsonParsed, err := gabs.ParseJSON([]byte(probe))
	if err != nil {
		return err
	}

	f.JSON = *jsonParsed
	return nil
}

func (f *FFMpegHelper) GetLength() (float64, error) {
	s := f.JSON.Path("format.duration").Data().(string)
	d, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return d, nil
}
