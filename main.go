package main

import (
	"encoding/json"
	"net/http"
	"regexp"

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

	for _, stream := range streams {
		stream.StartReplay()
	}

	log.Info("Started replay for all streams")

	http.HandleFunc("/{server}/{key}", func(wr http.ResponseWriter, req *http.Request) {
		server := req.PathValue("server")
		key := req.PathValue("key")

		rgx := regexp.MustCompile(`(?m)^(c\d{1,2})s\d{1,2}$`)
		matches := rgx.FindStringSubmatch(server)
		if len(matches) < 2 {
			abort(wr, http.StatusBadRequest, "invalid server")

			return
		}

		key = matches[1] + ":" + key

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
