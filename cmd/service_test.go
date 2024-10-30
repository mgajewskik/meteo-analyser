package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func mockHTTPClient(handler http.HandlerFunc) *http.Client {
	server := httptest.NewServer(handler)
	return &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(server.URL)
			},
		},
		Timeout: 10 * time.Second,
	}
}

func TestService_CollectResponses(t *testing.T) {
	tests := []struct {
		name       string
		cities     []City
		mockClient *http.Client
		expectErr  bool
	}{
		{
			name: "Successful data collection",
			cities: []City{
				{City: "CityA", Lat: "10.0", Lng: "20.0"},
				{City: "CityB", Lat: "15.0", Lng: "25.0"},
			},
			mockClient: mockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(
					w,
					`{"daily": {"temperature_2m_mean": [20.0, 25.0, 30.0], "weather_code": [1, 2, 1, 3, 1]}}`,
				)
			}),
			expectErr: false,
		},
		{
			name: "Error during data fetch",
			cities: []City{
				{City: "CityA", Lat: "10.0", Lng: "20.0"},
			},
			mockClient: mockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `Internal Server Error`)
			}),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config{
				inputFile:        "test_input.json",
				bufferSize:       10,
				numWorkers:       2,
				startDate:        "2023-01-01",
				endDate:          "2023-01-07",
				baseOpenMeteoURL: "http://mockserver",
				mistCode:         1,
				clearSkyCode:     2,
			}

			svc := &service{
				config:     cfg,
				httpClient: tt.mockClient,
			}

			inputFile, err := os.CreateTemp("", "test_input.json")
			assert.NoError(t, err)
			defer os.Remove(inputFile.Name())

			data, err := json.Marshal(tt.cities)
			assert.NoError(t, err)
			_, err = inputFile.Write(data)
			assert.NoError(t, err)
			inputFile.Close()

			svc.config.inputFile = inputFile.Name()

			svc.CollectResponses()

			if !tt.expectErr {
				assert.Equal(t, len(tt.cities), len(svc.responses))
			}
		})
	}
}

func TestService_AnalyseResponses(t *testing.T) {
	tests := []struct {
		name      string
		responses []CityResponse
		expected  Results
		expectErr bool
	}{
		{
			name: "Successful analysis",
			responses: []CityResponse{
				{CityName: "CityA", MeanTemperature: 20.0, DaysWithMist: 3, DaysWithClearSky: 5},
				{CityName: "CityB", MeanTemperature: 25.0, DaysWithMist: 2, DaysWithClearSky: 6},
			},
			expected: Results{
				HighestMeanTemperatureCity: "CityB",
				MostMistCity:               "CityA",
				MostClearSkyCity:           "CityB",
			},
			expectErr: false,
		},
		{
			name:      "No responses to analyse",
			responses: nil,
			expected:  Results{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config{
				mistCode:     1,
				clearSkyCode: 2,
			}

			svc := &service{
				config:    cfg,
				responses: tt.responses,
			}

			err := svc.AnalyseResponses()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, svc.results)
			}
		})
	}
}

func TestService_ProcessCity(t *testing.T) {
	tests := []struct {
		name       string
		city       City
		mockClient *http.Client
		expected   CityResponse
	}{
		{
			name: "Successful data fetch",
			city: City{City: "CityA", Lat: "10.0", Lng: "20.0"},
			mockClient: mockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintln(
					w,
					`{"daily": {"temperature_2m_mean": [20.0, 25.0, 30.0], "weather_code": [1, 2, 1, 3, 1]}}`,
				)
			}),
			expected: CityResponse{
				CityName:         "CityA",
				MeanTemperature:  25.0,
				DaysWithMist:     3,
				DaysWithClearSky: 1,
				Error:            nil,
			},
		},
		{
			name: "Error during data fetch",
			city: City{City: "CityB", Lat: "error", Lng: "20.0"},
			mockClient: mockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintln(w, `Internal Server Error`)
			}),
			expected: CityResponse{
				CityName: "CityB",
				Error: fmt.Errorf(
					"failed to make request: %w",
					fmt.Errorf("failed to make request: %w", fmt.Errorf("Internal Server Error")),
				),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config{
				startDate:        "2023-01-01",
				endDate:          "2023-01-07",
				baseOpenMeteoURL: "http://mockserver",
				mistCode:         1,
				clearSkyCode:     2,
			}

			svc := &service{
				config:     cfg,
				httpClient: tt.mockClient,
			}

			result := svc.processCity(tt.city)

			assert.Equal(t, tt.expected.CityName, result.CityName)
			assert.Equal(t, tt.expected.MeanTemperature, result.MeanTemperature)
			assert.Equal(t, tt.expected.DaysWithMist, result.DaysWithMist)
			assert.Equal(t, tt.expected.DaysWithClearSky, result.DaysWithClearSky)
			if tt.expected.Error != nil {
				assert.Error(t, result.Error)
			} else {
				assert.NoError(t, result.Error)
			}
		})
	}
}

func TestCalculateMeanTemperature(t *testing.T) {
	tests := []struct {
		name     string
		temps    []float64
		expected float64
	}{
		{
			name:     "Empty slice",
			temps:    []float64{},
			expected: 0,
		},
		{
			name:     "One temperature",
			temps:    []float64{25.0},
			expected: 25.0,
		},
		{
			name:     "Multiple temperatures",
			temps:    []float64{20.0, 25.0, 30.0},
			expected: 25.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateMeanTemperature(tt.temps)
			if result != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, result)
			}
		})
	}
}

func TestCountWeatherCodes(t *testing.T) {
	tests := []struct {
		name         string
		weatherCodes []int
		desiredCode  int
		expected     int
	}{
		{
			name:         "No weather codes",
			weatherCodes: []int{},
			desiredCode:  1,
			expected:     0,
		},
		{
			name:         "Some weather codes matching",
			weatherCodes: []int{1, 2, 1, 3, 1},
			desiredCode:  1,
			expected:     3,
		},
		{
			name:         "All weather codes matching",
			weatherCodes: []int{1, 1, 1, 1},
			desiredCode:  1,
			expected:     4,
		},
		{
			name:         "No weather codes matching",
			weatherCodes: []int{2, 3, 4, 5},
			desiredCode:  1,
			expected:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countWeatherCodes(tt.weatherCodes, tt.desiredCode)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}
