const state = {
  currentId: null,
  notes: [],
  previewTimer: null,
};

const $ = (sel) => document.querySelector(sel);

const els = {
  notesList: $("#notes-list"),
  notesEmpty: $("#notes-empty"),
  title: $("#note-title"),
  content: $("#note-content"),
  preview: $("#preview"),
  grammarResults: $("#grammar-results"),
  editorStatus: $("#editor-status"),
  toast: $("#toast"),
  fileUpload: $("#file-upload"),
};

function showToast(msg, isError = false) {
  els.toast.textContent = msg;
  els.toast.classList.toggle("error", isError);
  els.toast.classList.remove("hidden");
  clearTimeout(showToast._t);
  showToast._t = setTimeout(() => els.toast.classList.add("hidden"), 3200);
}

function setStatus(text) {
  els.editorStatus.textContent = text;
}

async function api(path, options = {}) {
  const res = await fetch(path, {
    headers: { "Content-Type": "application/json", ...options.headers },
    ...options,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || res.statusText || "Request failed");
  }
  return data;
}

function formatDate(iso) {
  try {
    return new Date(iso).toLocaleString(undefined, {
      dateStyle: "short",
      timeStyle: "short",
    });
  } catch {
    return iso;
  }
}

async function loadNotesList() {
  const notes = await api("/api/notes");
  state.notes = notes;
  els.notesList.innerHTML = "";

  if (!notes.length) {
    els.notesEmpty.classList.remove("hidden");
    return;
  }
  els.notesEmpty.classList.add("hidden");

  notes.forEach((note) => {
    const li = document.createElement("li");
    const btn = document.createElement("button");
    btn.type = "button";
    btn.dataset.id = note.id;
    if (note.id === state.currentId) btn.classList.add("active");
    btn.innerHTML = `${escapeHTML(note.title || "Untitled")}<span class="note-item-date">${formatDate(note.created_at)}</span>`;
    btn.addEventListener("click", () => openNote(note.id));
    li.appendChild(btn);
    els.notesList.appendChild(li);
  });
}

async function openNote(id) {
  const note = await api(`/api/notes/${id}`);
  state.currentId = note.id;
  els.title.value = note.title || "";
  els.content.value = note.content || "";
  setStatus(`Editing · ${note.id.slice(0, 8)}…`);
  highlightActiveNote(id);
  schedulePreview();
}

function highlightActiveNote(id) {
  els.notesList.querySelectorAll("button").forEach((btn) => {
    btn.classList.toggle("active", btn.dataset.id === id);
  });
}

function newNote() {
  state.currentId = null;
  els.title.value = "";
  els.content.value = "# New Note\n\n";
  els.preview.innerHTML = "";
  els.grammarResults.innerHTML =
    '<p class="grammar-hint">Run a grammar check on the current note content.</p>';
  setStatus("New note (unsaved)");
  highlightActiveNote(null);
  els.content.focus();
}

async function saveNote() {
  const title = els.title.value.trim();
  const content = els.content.value;
  if (!content.trim()) {
    showToast("Note content cannot be empty", true);
    return;
  }

  try {
    if (state.currentId) {
      const note = await api(`/api/notes/${state.currentId}`, {
        method: "PUT",
        body: JSON.stringify({ title, content }),
      });
      setStatus(`Saved · ${formatDate(note.created_at)}`);
      showToast("Note updated");
    } else {
      const note = await api("/api/notes", {
        method: "POST",
        body: JSON.stringify({ title, content }),
      });
      state.currentId = note.id;
      setStatus(`Created · ${note.id.slice(0, 8)}…`);
      showToast("Note saved");
    }
    await loadNotesList();
    highlightActiveNote(state.currentId);
  } catch (err) {
    showToast(err.message, true);
  }
}

async function refreshPreview() {
  const content = els.content.value;
  if (!content.trim()) {
    els.preview.innerHTML = "<p><em>Nothing to preview</em></p>";
    return;
  }

  const data = await api("/api/preview", {
    method: "POST",
    body: JSON.stringify({ content }),
  });
  els.preview.innerHTML = data.html;
}

function schedulePreview() {
  clearTimeout(state.previewTimer);
  state.previewTimer = setTimeout(() => refreshPreview().catch(() => {}), 400);
}

async function checkGrammar() {
  const content = els.content.value;
  if (!content.trim()) {
    showToast("Add some content first", true);
    return;
  }

  switchTab("grammar");
  els.grammarResults.innerHTML = "<p class=\"grammar-hint\">Checking…</p>";

  try {
    const result = await api("/api/grammar/check", {
      method: "POST",
      body: JSON.stringify({
        content,
        strip_markdown: true,
        language: "en-US",
      }),
    });

    if (!result.matches || result.matches.length === 0) {
      els.grammarResults.innerHTML =
        '<p class="grammar-ok">No issues found.</p>';
      showToast("Grammar check passed");
      return;
    }

    els.grammarResults.innerHTML = result.matches
      .map(
        (m) => `
      <div class="grammar-match">
        <strong>${escapeHTML(m.message)}</strong>
        ${m.context ? `<div>${escapeHTML(m.context)}</div>` : ""}
        ${
          m.suggestions?.length
            ? `<div class="suggestions">Suggestions: ${m.suggestions.map(escapeHTML).join(", ")}</div>`
            : ""
        }
      </div>`
      )
      .join("");
    showToast(`${result.matches.length} issue(s) found`);
  } catch (err) {
    els.grammarResults.innerHTML = `<p class="grammar-hint">${escapeHTML(err.message)}</p>`;
    showToast(err.message, true);
  }
}

function switchTab(name) {
  document.querySelectorAll(".tab").forEach((t) => {
    t.classList.toggle("active", t.dataset.tab === name);
  });
  document.querySelectorAll(".tab-pane").forEach((p) => {
    p.classList.toggle("active", p.id === `tab-${name}`);
  });
}

function escapeHTML(s) {
  const d = document.createElement("div");
  d.textContent = s;
  return d.innerHTML;
}

async function handleFileUpload(file) {
  const form = new FormData();
  form.append("file", file);
  if (els.title.value.trim()) form.append("title", els.title.value.trim());

  const res = await fetch("/api/notes", { method: "POST", body: form });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Upload failed");

  state.currentId = data.id;
  await openNote(data.id);
  showToast(`Uploaded ${file.name}`);
  await loadNotesList();
}

$("#btn-new").addEventListener("click", newNote);
$("#btn-save").addEventListener("click", () => saveNote());
$("#btn-grammar").addEventListener("click", () => checkGrammar());
$("#btn-refresh").addEventListener("click", () =>
  loadNotesList().catch((e) => showToast(e.message, true))
);

els.content.addEventListener("input", schedulePreview);
els.title.addEventListener("input", schedulePreview);

document.querySelectorAll(".tab").forEach((tab) => {
  tab.addEventListener("click", () => switchTab(tab.dataset.tab));
});

els.fileUpload.addEventListener("change", async (e) => {
  const file = e.target.files?.[0];
  if (!file) return;
  try {
    await handleFileUpload(file);
  } catch (err) {
    showToast(err.message, true);
  }
  e.target.value = "";
});

loadNotesList()
  .then(() => {
    if (!state.notes.length) newNote();
  })
  .catch((e) => showToast(e.message, true));
