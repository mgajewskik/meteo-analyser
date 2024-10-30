package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
)

type service struct {
	config     config
	httpClient *http.Client
	responses  []CityResponse
	results    Results
	count      int
}

func (s *service) CollectResponses() {
	cities := make(chan City, s.config.bufferSize)
	cityResults := make(chan CityResponse, s.config.bufferSize)

	go func() {
		defer close(cities)
		if err := s.streamJSONToChannel(cities); err != nil {
			log.Printf("Error reading JSON: %v", err)
			return
		}
	}()

	var wg sync.WaitGroup
	wg.Add(s.config.numWorkers)

	for i := 0; i < s.config.numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()
			for city := range cities {
				cityResults <- s.processCity(city)
				// NOTE: comment this out to shorten the output
				log.Printf("Worker %d processed city: %s", workerID, city.City)
			}
		}(i)
	}

	go func() {
		wg.Wait()
		close(cityResults)
	}()

	for r := range cityResults {
		s.responses = append(s.responses, r)
	}
}

func (s *service) streamJSONToChannel(cities chan<- City) error {
	file, err := os.Open(s.config.inputFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	_, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read JSON token: %w", err)
	}

	for decoder.More() {
		var city City
		if err := decoder.Decode(&city); err != nil {
			return fmt.Errorf("failed to decode city: %w", err)
		}
		cities <- city
	}

	return nil
}

func (s *service) processCity(city City) CityResponse {
	resp, err := s.fetchWeatherData(city.Lat, city.Lng, s.config.startDate, s.config.endDate)
	if err != nil {
		return CityResponse{CityName: city.City, Error: err}
	}

	return CityResponse{
		CityName:         city.City,
		MeanTemperature:  calculateMeanTemperature(resp.Daily.Temperature2mMean),
		DaysWithMist:     countWeatherCodes(resp.Daily.WeatherCode, s.config.mistCode),
		DaysWithClearSky: countWeatherCodes(resp.Daily.WeatherCode, s.config.clearSkyCode),
		Error:            nil,
	}
}

func (s *service) fetchWeatherData(
	latitude, longitude,
	startDate, endDate string,
) (*WeatherResponse, error) {
	params := url.Values{}
	params.Add("latitude", latitude)
	params.Add("longitude", longitude)
	params.Add("start_date", startDate)
	params.Add("end_date", endDate)
	params.Add("daily", "weather_code,temperature_2m_mean")
	params.Add("timezone", "Europe/Berlin")

	fullURL := fmt.Sprintf("%s?%s", s.config.baseOpenMeteoURL, params.Encode())

	resp, err := s.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status: %d, body: %s", resp.StatusCode, body)
	}

	var weather WeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&weather); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &weather, nil
}

func (s *service) AnalyseResponses() error {
	if s.responses == nil {
		return fmt.Errorf("no responses to analyse, run the collection step first")
	}

	var (
		highestMeanTemp float64
		mostMist        int
		mostClearSky    int
	)

	for _, r := range s.responses {
		if r.Error != nil {
			log.Printf("Error processing city %s: %v", r.CityName, r.Error)
			continue
		}

		if r.MeanTemperature > highestMeanTemp {
			highestMeanTemp = r.MeanTemperature
			s.results.HighestMeanTemperatureCity = r.CityName
		}

		if r.DaysWithMist > mostMist {
			mostMist = r.DaysWithMist
			s.results.MostMistCity = r.CityName
		}

		if r.DaysWithClearSky > mostClearSky {
			mostClearSky = r.DaysWithClearSky
			s.results.MostClearSkyCity = r.CityName
		}

		s.count++
	}

	if mostMist == 0 && s.results.MostMistCity == "" {
		s.results.MostMistCity = "No city had mist"
	}

	if mostClearSky == 0 && s.results.MostClearSkyCity == "" {
		s.results.MostClearSkyCity = "No city had clear sky"
	}

	return nil
}

func calculateMeanTemperature(temps []float64) float64 {
	if len(temps) == 0 {
		return 0
	}

	var sum float64
	for _, temp := range temps {
		sum += temp
	}

	return sum / float64(len(temps))
}

func countWeatherCodes(weatherCodes []int, desiredCode int) int {
	var count int
	for _, code := range weatherCodes {
		if code == desiredCode {
			count++
		}
	}

	return count
}
