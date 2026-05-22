package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/thefool/markdown-notes/internal/grammar"
	"github.com/thefool/markdown-notes/internal/handler"
	"github.com/thefool/markdown-notes/internal/store"
)

//go:embed all:static
var webStatic embed.FS

func main() {
	port := envOrDefault("PORT", "8080")
	notesDir := envOrDefault("NOTES_DIR", "notes")
	ltURL := os.Getenv("LANGUAGETOOL_URL")

	noteStore, err := store.NewFileStore(notesDir)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}

	h := handler.New(noteStore, grammar.NewChecker(ltURL))

	staticFS, err := fs.Sub(webStatic, "static")
	if err != nil {
		log.Fatalf("static files: %v", err)
	}

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
		r.Put("/notes/{id}", h.UpdateNote)
		r.Get("/notes/{id}/html", h.RenderNoteHTML)
		r.Post("/preview", h.PreviewMarkdown)
		r.Post("/grammar/check", h.CheckGrammar)
	})

	r.Get("/", serveStatic(staticFS, "index.html"))
	r.Get("/css/*", serveStaticFile(staticFS))
	r.Get("/js/*", serveStaticFile(staticFS))

	log.Printf("GUI available at http://localhost:%s", port)
	log.Printf("listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func serveStatic(staticFS fs.FS, file string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, staticFS, file)
	}
}

func serveStaticFile(staticFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || strings.Contains(path, "..") {
			http.NotFound(w, r)
			return
		}
		http.ServeFileFS(w, r, staticFS, path)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
