package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

type config struct {
	baseOpenMeteoURL string
	startDate        string
	endDate          string
	mistCode         int
	clearSkyCode     int
	inputFile        string
	outputFile       string
	numWorkers       int
	bufferSize       int
}

func newConfig() config {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	inputFile := flag.String("input",
		filepath.Join(dir, "data/pl2.json"),
		"Input JSON file path")

	outputFile := flag.String("output",
		filepath.Join(dir, "data/results.json"),
		"Output JSON file path")

	numWorkers := flag.Int("workers",
		1,
		"Number of worker goroutines")

	bufferSize := flag.Int("buffer",
		200,
		"Channel buffer size")

	flag.Parse()

	return config{
		baseOpenMeteoURL: "https://archive-api.open-meteo.com/v1/archive",
		startDate:        "2024-04-01",
		endDate:          "2024-09-30",
		mistCode:         45,
		clearSkyCode:     0,
		inputFile:        *inputFile,
		outputFile:       *outputFile,
		numWorkers:       *numWorkers,
		bufferSize:       *bufferSize,
	}
}
