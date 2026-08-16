package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/nazboyko/good-dog/internal/animal"
	"github.com/nazboyko/good-dog/internal/audiocache"
	"github.com/nazboyko/good-dog/internal/config"
	"github.com/nazboyko/good-dog/internal/dogsheet"
	"github.com/nazboyko/good-dog/internal/elevenlabs"
	"github.com/nazboyko/good-dog/internal/gemini"
	"github.com/nazboyko/good-dog/internal/httpapi"
	"github.com/nazboyko/good-dog/internal/radio"
	"github.com/nazboyko/good-dog/internal/session"
)

// a life untouched this long is over, the row is purged at startup
const sessionTTL = 48 * time.Hour

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("load .env: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Take the port before doing anything else.
	//
	// ListenAndServe at the bottom of main means a stale process holding
	// the port is the last line of a startup log that otherwise looks
	// completely healthy: the preflight passes, the voice reports ready,
	// and then the process dies while the old binary keeps answering.
	// Every symptom after that belongs to code that is not running. That
	// cost an hour, four times, in two days.
	//
	// Failing here makes it the first line instead of the last.
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("cannot take port %s, something else is already on it: %v", port, err)
	}
	log.Printf("holding :%s, starting up", port)

	var geminiClient *gemini.Client
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		geminiClient = gemini.New(key)
		preflightCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		geminiClient.PreflightModel(preflightCtx)
		cancel()
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

	provider, err := animal.NewFixtureProvider("fixtures/dogs.json")
	if err != nil {
		log.Fatalf("load fixtures: %v", err)
	}
	// a typed nil must not sneak into the interface, the compiler checks
	// for a nil Generator and serves the default sheet
	var llm dogsheet.Generator
	if geminiClient != nil {
		llm = geminiClient
	}
	compiler := dogsheet.NewCompiler(llm, dogsheet.NewCache("cache/sheets"))
	// sessions survive a restart: sqlite is the truth, the store caches it.
	// A life left open for days is not resumed, the player gets a clean one.
	sessionDB, err := session.OpenDB("cache/sessions.db")
	if err != nil {
		log.Fatalf("open sessions db: %v", err)
	}
	if n, err := sessionDB.Purge(context.Background(), time.Now().Add(-sessionTTL)); err != nil {
		log.Printf("purge sessions: %v", err)
	} else if n > 0 {
		log.Printf("purged %d stale sessions", n)
	}
	// short rails are the playtest, RUN_RAILS=full walks the three days
	rails := session.ShortRun
	if os.Getenv("RUN_RAILS") == "full" {
		rails = session.FullRun
	}
	// FIRST_DOG pins one dog for a playtest. Unset, which is the default
	// and how it ships, every run is a different real dog from the pool.
	sessions := httpapi.NewSessions(provider, compiler, sessionDB, rails, os.Getenv("FIRST_DOG"))

	// Ranger's voice, prepared before the door opens. The host says the
	// same three lines every night, so this is a one time cost that a
	// warm cache turns into nothing. Nothing is ever synthesized while a
	// player is listening: see internal/radio/voice.go.
	if elevenClient != nil {
		host := httpapi.NewHostVoice(ttsCache, elevenClient, &elevenlabs.Budget{},
			elevenlabs.DefaultVoiceID, elevenlabs.VoiceSettings{Stability: 0.5, SimilarityBoost: 0.75})
		if report, err := radio.Prepare(context.Background(), host); err != nil {
			log.Printf("radio: host voice not ready, tonight is text only: %v", err)
		} else {
			log.Print(report)
			if missing := radio.Missing(host); len(missing) > 0 {
				log.Printf("radio: %d host lines still missing after preparation: %v", len(missing), missing)
			} else {
				sessions = sessions.WithVoice(host)
			}
		}
	} else {
		log.Print("radio: no speech key, tonight is text only")
	}

	mux := http.NewServeMux()
	sessions.Register(mux)
	mux.HandleFunc("GET /events", httpapi.Events(sessions))
	mux.HandleFunc("GET /api/audio/{file}", httpapi.Audio("assets/audio", ttsCacheDir))
	mux.HandleFunc("GET /api/spike/gemini", httpapi.SpikeGemini(geminiClient))
	mux.HandleFunc("GET /api/spike/tts", httpapi.SpikeTTS(elevenClient, ttsCache, &elevenlabs.Budget{}))
	mux.HandleFunc("GET /api/spike/subscription", httpapi.SpikeSubscription(elevenClient))

	log.Printf("listening on :%s", port)
	if err := http.Serve(listener, mux); err != nil {
		log.Fatal(err)
	}
}
