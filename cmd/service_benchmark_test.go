package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// NOTE: those benchmarks use real data and API calls

func BenchmarkCollectResponses_Workers(b *testing.B) {
	workers := []int{1, 2, 3, 4, 6, 8, 10, 12, 14, 16, 20}
	for _, w := range workers {
		b.Run(fmt.Sprintf("%d-workers\n", w), func(b *testing.B) {
			cfg := config{
				baseOpenMeteoURL: "https://archive-api.open-meteo.com/v1/archive",
				startDate:        "2023-04-01",
				endDate:          "2023-09-30",
				mistCode:         45,
				clearSkyCode:     0,
				inputFile:        "../data/pl172.json",
				numWorkers:       w,
				bufferSize:       200,
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				svc := &service{
					config: cfg,
					httpClient: &http.Client{
						Timeout: 10 * time.Second,
					},
				}

				svc.CollectResponses()
				svc.AnalyseResponses()
			}
		})
	}
}
