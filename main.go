package main

import (
	"encoding/json"
	"net/http"

	"github.com/coalaura/logger"
)

var log = logger.New()

func main() {
	log.Info("Loading config...")

	cfg, err := LoadConfig()
	log.MustPanic(err)

	streams, err := FindStreams(cfg)
	log.MustPanic(err)

	if len(streams) == 0 {
		log.Fatal("No streams found")

		return
	}

	log.Infof("Found %d streams\n", len(streams))

	for i, stream := range streams {
		log.Debugf("Starting stream %s...\n", i)

		err = stream.StartReplay()
		log.MustPanic(err)
	}

	log.Info("Started replay for all streams")

	http.HandleFunc("/{key}", func(wr http.ResponseWriter, req *http.Request) {
		key := req.PathValue("key")

		stream, ok := streams[key]
		if !ok {
			abort(wr, http.StatusNotFound, "stream not found")

			return
		}

		if err := stream.Capture(wr); err != nil {
			abort(wr, http.StatusInternalServerError, err.Error())
		}
	})

	log.Fatal(http.ListenAndServe(":4644", nil))
}

func abort(wr http.ResponseWriter, code int, message string) {
	b, _ := json.Marshal(map[string]string{
		"error": message,
	})

	wr.Header().Set("Content-Type", "application/json")
	wr.WriteHeader(code)

	wr.Write(b)
}
