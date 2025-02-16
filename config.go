package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Panel    string `json:"panel"`
	Interval int    `json:"interval"`
	Duration int    `json:"duration"`
}

func LoadConfig() (*Config, error) {
	b, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}

	var config Config

	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func FindStreams(cfg *Config) (map[string]*Stream, error) {
	directory := filepath.Join(cfg.Panel, "envs")

	clusters, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	streams := make(map[string]*Stream)

	for _, cluster := range clusters {
		if !cluster.IsDir() {
			continue
		}

		name := cluster.Name()
		path := filepath.Join(directory, name, ".env")

		file, err := os.OpenFile(path, os.O_RDONLY, 0644)
		if err != nil {
			log.Warningf("Failed to read cluster %s: %s\n", name, err)

			continue
		}

		defer file.Close()

		env, err := godotenv.Parse(file)
		if err != nil {
			log.Warningf("Failed to parse cluster %s: %s\n", name, err)

			continue
		}

		url, ok := env["OVERWATCH_URL"]
		if !ok || url == "" {
			continue
		}

		raw, ok := env["OVERWATCH_STREAMS"]
		if !ok || raw == "" {
			continue
		}

		entries := strings.Split(raw, ",")

		for _, entry := range entries {
			entry = strings.TrimSpace(entry)

			parts := strings.SplitN(entry, ":", 2)
			if len(parts) != 2 {
				continue
			}

			key := parts[1]

			if _, ok := streams[key]; ok {
				log.Notef("Skipped duplicate stream %s for cluster %s\n", key, name)

				continue
			}

			stream, err := NewStream(cfg, key, fmt.Sprintf(url, key))
			if err != nil {
				log.Warningf("Failed to create stream %s for cluster %s: %s\n", key, name, err)

				continue
			}

			streams[key] = stream

			log.Notef("Found stream %s for cluster %s\n", key, name)
		}
	}

	return streams, nil
}
