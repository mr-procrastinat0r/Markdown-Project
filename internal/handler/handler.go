package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/thefool/markdown-notes/internal/grammar"
	"github.com/thefool/markdown-notes/internal/markdown"
	"github.com/thefool/markdown-notes/internal/store"
)

const maxBodySize = 10 << 20 // 10 MB

type Handler struct {
	store   *store.FileStore
	grammar *grammar.Checker
}

func New(s *store.FileStore, g *grammar.Checker) *Handler {
	return &Handler{store: s, grammar: g}
}

type createNoteJSON struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type grammarCheckJSON struct {
	Text     string `json:"text"`
	Content  string `json:"content"`
	Language string `json:"language"`
	Markdown bool   `json:"strip_markdown"`
}

func (h *Handler) CreateNote(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	var title, content string
	var err error

	if strings.HasPrefix(contentType, "multipart/form-data") {
		title, content, err = h.parseMultipartNote(r)
	} else {
		title, content, err = h.parseJSONNote(w, r)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if strings.TrimSpace(content) == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	note, err := h.store.Save(title, content)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save note")
		return
	}

	w.Header().Set("Location", "/api/notes/"+note.ID)
	writeJSON(w, http.StatusCreated, note)
}

func (h *Handler) UpdateNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	defer r.Body.Close()

	var req createNoteJSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	note, err := h.store.Update(id, req.Title, req.Content)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update note")
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (h *Handler) parseJSONNote(w http.ResponseWriter, r *http.Request) (string, string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	defer r.Body.Close()

	var req createNoteJSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", "", errors.New("invalid JSON body")
	}
	return req.Title, req.Content, nil
}

func (h *Handler) parseMultipartNote(r *http.Request) (string, string, error) {
	if err := r.ParseMultipartForm(maxBodySize); err != nil {
		return "", "", errors.New("invalid multipart form")
	}

	title := r.FormValue("title")

	file, header, err := r.FormFile("file")
	if err == nil {
		defer file.Close()
		if header.Size > maxBodySize {
			return "", "", errors.New("file too large")
		}
		data, err := io.ReadAll(io.LimitReader(file, maxBodySize+1))
		if err != nil {
			return "", "", errors.New("failed to read uploaded file")
		}
		if int64(len(data)) > maxBodySize {
			return "", "", errors.New("file too large")
		}
		if title == "" {
			title = strings.TrimSuffix(header.Filename, ".md")
			title = strings.TrimSuffix(title, ".markdown")
		}
		return title, string(data), nil
	}

	content := r.FormValue("content")
	if content == "" {
		return "", "", errors.New("file or content field is required")
	}
	return title, content, nil
}

func (h *Handler) ListNotes(w http.ResponseWriter, r *http.Request) {
	notes, err := h.store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list notes")
		return
	}
	writeJSON(w, http.StatusOK, notes)
}

func (h *Handler) GetNote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	note, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get note")
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (h *Handler) RenderNoteHTML(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	note, err := h.store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get note")
		return
	}

	body, err := markdown.ToHTML([]byte(note.Content))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to render markdown")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "json" {
		writeJSON(w, http.StatusOK, map[string]string{
			"id":    note.ID,
			"title": note.Title,
			"html":  string(body),
		})
		return
	}

	doc := markdown.WrapDocument(note.Title, body)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc)
}

type previewJSON struct {
	Content string `json:"content"`
}

func (h *Handler) PreviewMarkdown(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	defer r.Body.Close()

	var req previewJSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}

	body, err := markdown.ToHTML([]byte(req.Content))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to render markdown")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"html": string(body)})
}

func (h *Handler) CheckGrammar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)
	defer r.Body.Close()

	var req grammarCheckJSON
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	text := req.Text
	if text == "" {
		text = req.Content
	}
	if strings.TrimSpace(text) == "" {
		writeError(w, http.StatusBadRequest, "text or content is required")
		return
	}

	if req.Markdown {
		text = grammar.StripMarkdown(text)
	}

	result, err := h.grammar.Check(r.Context(), text, req.Language)
	if err != nil {
		writeError(w, http.StatusBadGateway, "grammar check failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
