# Markdown Note-taking App

A REST API and web GUI written in Go for saving Markdown notes, listing them, rendering HTML, and checking grammar.

## Features

- **Web GUI** — editor, live preview, note list, grammar panel, file upload
- **Save notes** — JSON body or multipart file upload
- **Update notes** — edit and save existing notes from the GUI
- **List notes** — metadata for all saved notes
- **Get note** — raw Markdown content
- **Render HTML** — GFM Markdown rendered to a full HTML page
- **Live preview** — `POST /api/preview` without saving
- **Grammar check** — LanguageTool HTTP API (plain text or Markdown-stripped)

## Requirements

- Go 1.22+
- Network access for grammar checks (uses the public LanguageTool API by default)

## Quick start

```bash
go mod tidy
go run ./cmd/server
```

Server listens on `http://localhost:8080` by default.

Open **http://localhost:8080** in your browser for the GUI.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `NOTES_DIR` | `notes` | Directory for persisted `.md` and metadata |
| `LANGUAGETOOL_URL` | LanguageTool public API | Grammar check endpoint |

## API

### Health

```bash
curl http://localhost:8080/health
```

### Save note (JSON)

```bash
curl -X POST http://localhost:8080/api/notes \
  -H "Content-Type: application/json" \
  -d '{"title":"My Note","content":"# Hello\n\nThis is **markdown**."}'
```

### Save note (file upload)

```bash
curl -X POST http://localhost:8080/api/notes \
  -F "title=Uploaded" \
  -F "file=@./example.md"
```

### List notes

```bash
curl http://localhost:8080/api/notes
```

### Get note (Markdown)

```bash
curl http://localhost:8080/api/notes/{id}
```

### Update note

```bash
curl -X PUT http://localhost:8080/api/notes/{id} \
  -H "Content-Type: application/json" \
  -d '{"title":"Updated","content":"# Revised\n\nNew body."}'
```

### Preview (no save)

```bash
curl -X POST http://localhost:8080/api/preview \
  -H "Content-Type: application/json" \
  -d '{"content":"# Hi\n\n**bold**"}'
```

### Render HTML

```bash
# Full HTML page
curl http://localhost:8080/api/notes/{id}/html

# JSON with html field
curl "http://localhost:8080/api/notes/{id}/html?format=json"
```

### Check grammar

```bash
curl -X POST http://localhost:8080/api/grammar/check \
  -H "Content-Type: application/json" \
  -d '{"text":"This are a mistake.","language":"en-US"}'
```

Check Markdown (strips syntax before checking):

```bash
curl -X POST http://localhost:8080/api/grammar/check \
  -H "Content-Type: application/json" \
  -d '{"content":"# Title\n\nThis are wrong.","strip_markdown":true}'
```

## Project layout

```
cmd/server/          # entrypoint (embeds static GUI)
cmd/server/static/   # GUI (HTML, CSS, JS)
internal/
  handler/           # HTTP handlers
  store/             # filesystem persistence
  markdown/          # goldmark rendering
  grammar/           # LanguageTool client
notes/               # saved notes (gitignored)
```

## Notes

- Grammar checking depends on [LanguageTool](https://languagetool.org/http-api/). The public API has rate limits; use a self-hosted instance via `LANGUAGETOOL_URL` for heavier use.
- Saved notes are stored as `{id}.md` with `{id}.json` metadata under `NOTES_DIR`.
