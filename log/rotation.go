package log

import (
	"io"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	lumberjackSchema = "lumberjack"
)

func init() {
	err := zap.RegisterSink(lumberjackSchema, func(u *url.URL) (zap.Sink, error) {
		r, err := rotationFromUrl(u)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshal rotation config")
		}
		return newSink(NewFileRotationWriter(*r)), nil
	})
	if err != nil {
		panic(err)
	}
}

type Rotation struct {
	File       string
	MaxSizeMb  int
	MaxDays    int
	MaxBackups int
	Compress   bool
}

func rotationToUrl(r Rotation) url.URL {
	values := url.Values{
		"file":       {r.File},
		"maxSizeMb":  {strconv.Itoa(r.MaxSizeMb)},
		"maxDays":    {strconv.Itoa(r.MaxDays)},
		"maxBackups": {strconv.Itoa(r.MaxBackups)},
		"compress":   {strconv.FormatBool(r.Compress)},
	}
	u := url.URL{
		Scheme:   lumberjackSchema,
		RawQuery: values.Encode(),
	}
	return u
}

func rotationFromUrl(u *url.URL) (*Rotation, error) {
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, errors.Wrap(err, "parse lumberjack params")
	}
	file := values.Get("file")
	maxSizeMb, err := strconv.Atoi(values.Get("maxSizeMb"))
	if err != nil {
		return nil, errors.Wrap(err, "parse maxSizeMb")
	}
	maxDays, err := strconv.Atoi(values.Get("maxDays"))
	if err != nil {
		return nil, errors.Wrap(err, "parse maxDays")
	}
	maxBackups, err := strconv.Atoi(values.Get("maxBackups"))
	if err != nil {
		return nil, errors.Wrap(err, "parse maxBackups")
	}
	compress, err := strconv.ParseBool(values.Get("compress"))
	if err != nil {
		return nil, errors.Wrap(err, "parse compress")
	}
	return &Rotation{
		File:       file,
		MaxSizeMb:  maxSizeMb,
		MaxDays:    maxDays,
		MaxBackups: maxBackups,
		Compress:   compress,
	}, nil
}

func NewFileRotationWriter(r Rotation) io.WriteCloser {
	return &lumberjack.Logger{
		Filename:   r.File,
		MaxSize:    r.MaxSizeMb,
		MaxAge:     r.MaxDays,
		MaxBackups: r.MaxBackups,
		Compress:   r.Compress,
	}
}

type sink struct {
	io.WriteCloser
}

func (s sink) Sync() error {
	return nil
}

func newSink(writer io.WriteCloser) sink {
	return sink{
		WriteCloser: writer,
	}
}
