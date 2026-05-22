package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("note not found")

type Note struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type NoteContent struct {
	Note
	Content string `json:"content"`
}

type meta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type FileStore struct {
	dir string
}

func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create notes directory: %w", err)
	}
	return &FileStore{dir: dir}, nil
}

func (s *FileStore) Save(title, content string) (Note, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	if title == "" {
		title = deriveTitle(content)
	}

	mdPath := s.mdPath(id)
	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		return Note{}, fmt.Errorf("write markdown: %w", err)
	}

	m := meta{ID: id, Title: title, CreatedAt: now}
	metaBytes, err := json.Marshal(m)
	if err != nil {
		_ = os.Remove(mdPath)
		return Note{}, fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(s.metaPath(id), metaBytes, 0644); err != nil {
		_ = os.Remove(mdPath)
		return Note{}, fmt.Errorf("write metadata: %w", err)
	}

	return Note{ID: id, Title: title, CreatedAt: now}, nil
}

func (s *FileStore) List() ([]Note, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("read notes directory: %w", err)
	}

	var notes []Note
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		m, err := s.loadMeta(strings.TrimSuffix(e.Name(), ".json"))
		if err != nil {
			continue
		}
		notes = append(notes, Note{ID: m.ID, Title: m.Title, CreatedAt: m.CreatedAt})
	}
	if notes == nil {
		notes = []Note{}
	}
	return notes, nil
}

func (s *FileStore) Get(id string) (NoteContent, error) {
	m, err := s.loadMeta(id)
	if err != nil {
		return NoteContent{}, err
	}
	content, err := os.ReadFile(s.mdPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return NoteContent{}, ErrNotFound
		}
		return NoteContent{}, fmt.Errorf("read markdown: %w", err)
	}
	return NoteContent{
		Note:    Note{ID: m.ID, Title: m.Title, CreatedAt: m.CreatedAt},
		Content: string(content),
	}, nil
}

func (s *FileStore) loadMeta(id string) (meta, error) {
	data, err := os.ReadFile(s.metaPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return meta{}, ErrNotFound
		}
		return meta{}, fmt.Errorf("read metadata: %w", err)
	}
	var m meta
	if err := json.Unmarshal(data, &m); err != nil {
		return meta{}, fmt.Errorf("parse metadata: %w", err)
	}
	return m, nil
}

func (s *FileStore) mdPath(id string) string {
	return filepath.Join(s.dir, id+".md")
}

func (s *FileStore) metaPath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func deriveTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return "Untitled"
}
