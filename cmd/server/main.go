package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/thefool/markdown-notes/internal/grammar"
	"github.com/thefool/markdown-notes/internal/handler"
	"github.com/thefool/markdown-notes/internal/store"
)

func main() {
	port := envOrDefault("PORT", "8080")
	notesDir := envOrDefault("NOTES_DIR", "notes")
	ltURL := os.Getenv("LANGUAGETOOL_URL")

	noteStore, err := store.NewFileStore(notesDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	h := handler.New(noteStore, grammar.NewChecker(ltURL))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api", func(r chi.Router) {
		r.Post("/notes", h.CreateNote)
		r.Get("/notes", h.ListNotes)
		r.Get("/notes/{id}", h.GetNote)
		r.Get("/notes/{id}/html", h.RenderNoteHTML)
		r.Post("/grammar/check", h.CheckGrammar)
	})

	log.Printf("listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
