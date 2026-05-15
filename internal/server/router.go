package server

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mojitrk/sica"
	"github.com/mojitrk/sica/internal/habits"
	"github.com/mojitrk/sica/internal/server/handler"
	"github.com/mojitrk/sica/internal/server/middleware"
	"github.com/mojitrk/sica/internal/store/sqlite"
	"github.com/mojitrk/sica/internal/tasks"
)

type Deps struct {
	Store        *sqlite.Store
	HabitsStore  *habits.Store
	TasksStore   *tasks.Store
}

func staticFS() (http.FileSystem, error) {
	sub, err := fs.Sub(sica.StaticFiles, "web/static")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

func New(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS)

	fs, err := staticFS()
	if err != nil {
		panic("failed to load static files: " + err.Error())
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(fs)))

	r.Get("/health", handler.Health)

	hhabits := &handler.HabitsHandler{Store: deps.HabitsStore}
	htasks := &handler.TasksHandler{Store: deps.TasksStore}

	r.Route("/api", func(r chi.Router) {
		r.Route("/habits", func(r chi.Router) {
			r.Get("/", hhabits.List)
			r.Post("/", hhabits.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", hhabits.Update)
				r.Delete("/", hhabits.Delete)
				r.Post("/entry", hhabits.SetEntry)
					r.Post("/increment", hhabits.Increment)
					r.Post("/decrement", hhabits.Decrement)
				r.Delete("/entry", hhabits.RemoveEntry)
				r.Get("/stats", hhabits.Stats)
			})
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", htasks.List)
			r.Post("/", htasks.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", htasks.Get)
				r.Put("/", htasks.Update)
				r.Delete("/", htasks.Delete)
				r.Post("/complete", htasks.Complete)
			})
		})

		r.Route("/projects", func(r chi.Router) {
			r.Get("/", htasks.ListProjects)
			r.Post("/", htasks.CreateProject)
		})

		r.Route("/transactions", func(r chi.Router) {
			r.Get("/", placeholder("list transactions"))
			r.Post("/", placeholder("create transaction"))
		})
		r.Get("/budgets", placeholder("list budgets"))
		r.Get("/summary", placeholder("financial summary"))

		r.Route("/calendar", func(r chi.Router) {
			r.Get("/events", placeholder("list events"))
			r.Post("/events", placeholder("create event"))
			r.Post("/sync-outlook", placeholder("sync outlook"))
		})

		r.Route("/knowledge", func(r chi.Router) {
			r.Post("/ingest", placeholder("ingest url"))
			r.Get("/search", placeholder("search knowledge"))
			r.Get("/docs", placeholder("list docs"))
			r.Get("/docs/{id}", placeholder("get doc"))
		})

		r.Route("/ai", func(r chi.Router) {
			r.Post("/chat", placeholder("ai chat"))
			r.Get("/conversations", placeholder("list conversations"))
		})
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(indexHTML))
	})

	return r
}

func placeholder(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"todo": msg})
	}
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>sica</title>
<link rel="stylesheet" href="/static/css/app.css">
</head>
<body>
<header><h1>sica</h1><span id="time"></span></header>
<nav>
  <a href="#" data-tab="habits">Habits</a>
  <a href="#" data-tab="tasks">Tasks</a>
  <a href="#" data-tab="finance">Finance</a>
  <a href="#" data-tab="calendar">Calendar</a>
  <a href="#" data-tab="knowledge">Knowledge</a>
  <a href="#" data-tab="chat">Chat</a>
</nav>
<main>
  <div id="habits" class="tab">
    <div id="habits-list"></div>
    <form id="habit-form" class="inline-form" style="margin-top:1rem">
      <input name="name" placeholder="New habit..." required>
      <select name="frequency"><option>daily</option><option>weekly</option></select>
      <button type="submit">Add</button>
    </form>
  </div>
  <div id="tasks" class="tab" hidden>
    <div id="tasks-list"></div>
    <form id="task-form" class="inline-form" style="margin-top:1rem">
      <input name="title" placeholder="New task..." required>
      <select name="priority"><option>med</option><option>high</option><option>low</option></select>
      <button type="submit">Add</button>
    </form>
  </div>
  <div id="finance" class="tab" hidden><div class="placeholder">Finance — coming soon</div></div>
  <div id="calendar" class="tab" hidden><div class="placeholder">Calendar — coming soon</div></div>
  <div id="knowledge" class="tab" hidden><div class="placeholder">Knowledge — coming soon</div></div>
  <div id="chat" class="tab" hidden><div class="placeholder">Chat — coming soon</div></div>
</main>
<script src="/static/js/app.js"></script>
<script>
document.querySelectorAll('nav a').forEach(a => {
  a.addEventListener('click', e => {
    e.preventDefault();
    document.querySelectorAll('nav a').forEach(l => l.classList.remove('active'));
    a.classList.add('active');
    document.querySelectorAll('.tab').forEach(t => t.hidden = true);
    document.getElementById(a.dataset.tab).hidden = false;
    if (a.dataset.tab === 'habits') loadHabits();
    if (a.dataset.tab === 'tasks') loadTasks();
  });
});
document.querySelector('nav a[data-tab="habits"]').classList.add('active');
setInterval(() => { document.getElementById('time').textContent = new Date().toLocaleTimeString(); }, 1000);

async function loadHabits() {
  const res = await fetch('/api/habits/');
  const habits = await res.json();
  const el = document.getElementById('habits-list');
  const today = new Date().toISOString().slice(0,10);
  if (!habits.length) { el.innerHTML = '<div class="placeholder">No habits yet</div>'; return; }
  el.innerHTML = habits.map(h => {
    const stats = '';
    return '<div class="card">' +
      '<div class="card-row"><strong>' + h.name + '</strong> <span class="dim">' + h.frequency + '</span></div>' +
      '<button class="btn-sm" onclick="completeHabit('+h.id+',\''+today+'\')">Done today</button>' +
      '</div>';
  }).join('');
}

async function completeHabit(id, date) {
  await fetch('/api/habits/'+id+'/entry', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({date: date, value: 1})
  });
  loadHabits();
}

document.getElementById('habit-form').addEventListener('submit', async e => {
  e.preventDefault();
  const fd = new FormData(e.target);
  await fetch('/api/habits/', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({name: fd.get('name'), frequency: fd.get('frequency')})
  });
  e.target.reset();
  loadHabits();
});

async function loadTasks() {
  const res = await fetch('/api/tasks/');
  const tasks = await res.json();
  const el = document.getElementById('tasks-list');
  if (!tasks.length) { el.innerHTML = '<div class="placeholder">No tasks yet</div>'; return; }
  el.innerHTML = tasks.map(t => {
    const done = t.status === 'done' ? ' class="done"' : '';
    return '<div class="card"><div class="card-row"' + done + '>' +
      '<span>[' + t.priority + ']</span> <strong>' + t.title + '</strong>' +
      (t.status !== 'done'
        ? ' <button class="btn-sm" onclick="completeTask('+t.id+')">Done</button>'
        : '') +
      '</div></div>';
  }).join('');
}

async function completeTask(id) {
  await fetch('/api/tasks/'+id+'/complete', {method:'POST'});
  loadTasks();
}

document.getElementById('task-form').addEventListener('submit', async e => {
  e.preventDefault();
  const fd = new FormData(e.target);
  await fetch('/api/tasks/', {
    method: 'POST',
    headers: {'Content-Type':'application/json'},
    body: JSON.stringify({title: fd.get('title'), priority: fd.get('priority')})
  });
  e.target.reset();
  loadTasks();
});

loadHabits();
</script>
</body>
</html>`
