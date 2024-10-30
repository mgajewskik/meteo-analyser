package main

type City struct {
	City             string `json:"city"`
	Lat              string `json:"lat"`
	Lng              string `json:"lng"`
	Country          string `json:"country"`
	ISO2             string `json:"iso2"`
	AdminName        string `json:"admin_name"`
	Capital          string `json:"capital"`
	Population       string `json:"population"`
	PopulationProper string `json:"population_proper"`
}

type CityResponse struct {
	CityName         string
	MeanTemperature  float64
	DaysWithMist     int
	DaysWithClearSky int
	Error            error
}

type WeatherResponse struct {
	Latitude         float64    `json:"latitude"`
	Longitude        float64    `json:"longitude"`
	GenerationTimeMs float64    `json:"generationtime_ms"`
	UTCOffsetSeconds int        `json:"utc_offset_seconds"`
	Timezone         string     `json:"timezone"`
	TimezoneAbbr     string     `json:"timezone_abbreviation"`
	Elevation        float64    `json:"elevation"`
	DailyUnits       DailyUnits `json:"daily_units"`
	Daily            Daily      `json:"daily"`
}

type DailyUnits struct {
	Time              string `json:"time"`
	WeatherCode       string `json:"weather_code"`
	Temperature2mMean string `json:"temperature_2m_mean"`
}

type Daily struct {
	Time              []string  `json:"time"`
	WeatherCode       []int     `json:"weather_code"`
	Temperature2mMean []float64 `json:"temperature_2m_mean"`
}

type Results struct {
	HighestMeanTemperatureCity string `json:"highest_mean_temperature_city"`
	MostMistCity               string `json:"most_mist_city"`
	MostClearSkyCity           string `json:"most_clear_sky_city"`
}
