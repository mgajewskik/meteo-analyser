package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	config := newConfig()

	log.Printf("Input file: %s", config.inputFile)
	log.Printf("Output file: %s", config.outputFile)
	log.Printf("Number of workers: %d", config.numWorkers)
	log.Printf("Buffer size: %d", config.bufferSize)

	svc := service{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	startTime := time.Now()

	svc.CollectResponses()

	err := svc.AnalyseResponses()
	if err != nil {
		log.Panic("Error analysing responses: %w", err)
	}

	if err := saveResults(svc.results, config.outputFile); err != nil {
		log.Printf("Error writing result: %v", err)
	}

	duration := time.Since(startTime)

	log.Printf("Processing completed")
	log.Printf("Total cities processed: %d", svc.count)
	log.Printf("Total execution time: %v", duration)
	log.Printf("Average time per city: %v", duration/time.Duration(svc.count))
}

func saveResults(result Results, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed to encode result: %w", err)
	}

	return nil
}
