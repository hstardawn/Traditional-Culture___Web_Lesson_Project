package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"traditional-culture-web-lesson/internal/travelagent"
)

func main() {
	port := envOrDefault("PORT", "8080")

	agent, err := travelagent.NewTravelAdvisorAgent(context.Background(), travelagent.AgentConfig{
		APIKey:  os.Getenv("SILICONFLOW_API_KEY"),
		Model:   os.Getenv("SILICONFLOW_MODEL"),
		BaseURL: os.Getenv("SILICONFLOW_BASE_URL"),
	})
	if err != nil {
		log.Fatalf("init travel advisor agent: %v", err)
	}

	mux := http.NewServeMux()
	travelagent.NewHandler(agent).Register(mux)
	mux.Handle("/", http.FileServer(http.Dir("..")))

	log.Printf("travel advisor server listening on http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func envOrDefault(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
