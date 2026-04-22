package travelagent

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AdviceRequest struct {
	Message     string        `json:"message"`
	Destination string        `json:"destination,omitempty"`
	TravelDate  string        `json:"travelDate,omitempty"`
	History     []ChatMessage `json:"history,omitempty"`
}

type TravelPlan struct {
	Destination        string   `json:"destination,omitempty"`
	TravelDate         string   `json:"travelDate,omitempty"`
	NeedsClarification []string `json:"needsClarification,omitempty"`
}

type WeatherContext struct {
	Available        bool    `json:"available"`
	Location         string  `json:"location,omitempty"`
	Date             string  `json:"date,omitempty"`
	Summary          string  `json:"summary"`
	TemperatureMin   float64 `json:"temperatureMin,omitempty"`
	TemperatureMax   float64 `json:"temperatureMax,omitempty"`
	PrecipitationMax float64 `json:"precipitationMax,omitempty"`
	WindSpeedMax     float64 `json:"windSpeedMax,omitempty"`
	Source           string  `json:"source"`
}

type AlmanacContext struct {
	Available bool     `json:"available"`
	Date      string   `json:"date,omitempty"`
	Yi        []string `json:"yi,omitempty"`
	Ji        []string `json:"ji,omitempty"`
	Note      string   `json:"note"`
	Source    string   `json:"source"`
}

type TravelContext struct {
	CurrentTime string         `json:"currentTime"`
	CurrentDate string         `json:"currentDate"`
	Timezone    string         `json:"timezone"`
	Plan        TravelPlan     `json:"plan"`
	Weather     WeatherContext `json:"weather"`
	Almanac     AlmanacContext `json:"almanac"`
	Risks       []string       `json:"risks"`
	FetchedAt   string         `json:"fetchedAt"`
}

type StreamEvent struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}
