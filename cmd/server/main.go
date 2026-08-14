package main

import (
	"log"
	"net/http"
	"os"

	"github.com/nazboyko/good-dog/internal/audiocache"
	"github.com/nazboyko/good-dog/internal/config"
	"github.com/nazboyko/good-dog/internal/elevenlabs"
	"github.com/nazboyko/good-dog/internal/gemini"
	"github.com/nazboyko/good-dog/internal/httpapi"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("load .env: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var geminiClient *gemini.Client
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		geminiClient = gemini.New(key, os.Getenv("GEMINI_MODEL"))
	} else {
		log.Print("GEMINI_API_KEY not set, gemini endpoints disabled")
	}

	var elevenClient *elevenlabs.Client
	if key := os.Getenv("ELEVENLABS_API_KEY"); key != "" {
		elevenClient = elevenlabs.New(key)
	} else {
		log.Print("ELEVENLABS_API_KEY not set, tts endpoints disabled")
	}
	// the audio route must serve where the tts cache writes
	const ttsCacheDir = "cache/audio"
	ttsCache := audiocache.New(ttsCacheDir)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /events", httpapi.Events())
	mux.HandleFunc("GET /api/audio/{file}", httpapi.Audio("assets/audio", ttsCacheDir))
	mux.HandleFunc("GET /api/spike/gemini", httpapi.SpikeGemini(geminiClient))
	mux.HandleFunc("GET /api/spike/tts", httpapi.SpikeTTS(elevenClient, ttsCache, &elevenlabs.Budget{}))

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
