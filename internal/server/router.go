package server

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mojitrk/sica"
	"github.com/mojitrk/sica/internal/server/handler"
	"github.com/mojitrk/sica/internal/server/middleware"
)

func staticFS() (http.FileSystem, error) {
	sub, err := fs.Sub(sica.StaticFiles, "web/static")
	if err != nil {
		return nil, err
	}
	return http.FS(sub), nil
}

func New() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.CORS)

	staticFS, err := staticFS()
	if err != nil {
		panic("failed to load static files: " + err.Error())
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(staticFS)))

	r.Get("/health", handler.Health)

	r.Route("/api", func(r chi.Router) {
		r.Route("/habits", func(r chi.Router) {
			r.Get("/", placeholder("list habits"))
			r.Post("/", placeholder("create habit"))
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", placeholder("update habit"))
				r.Delete("/", placeholder("delete habit"))
				r.Post("/entry", placeholder("add habit entry"))
				r.Get("/stats", placeholder("habit stats"))
			})
		})

		r.Route("/tasks", func(r chi.Router) {
			r.Get("/", placeholder("list tasks"))
			r.Post("/", placeholder("create task"))
			r.Route("/{id}", func(r chi.Router) {
				r.Put("/", placeholder("update task"))
				r.Delete("/", placeholder("delete task"))
			})
		})

		r.Route("/projects", func(r chi.Router) {
			r.Get("/", placeholder("list projects"))
			r.Post("/", placeholder("create project"))
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
		handler.Respond(w, http.StatusOK, map[string]string{"todo": msg})
	}
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>sica</title>
<style>
  :root { --bg: #1a1a2e; --fg: #e0e0e0; --accent: #7c9acc; --dim: #555; }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: ui-monospace, "Cascadia Code", "Fira Code", monospace; background: var(--bg); color: var(--fg); min-height: 100vh; }
  header { padding: 1rem; border-bottom: 1px solid var(--dim); display: flex; justify-content: space-between; align-items: center; }
  header h1 { font-size: 1.2rem; font-weight: normal; color: var(--accent); }
  nav { display: flex; gap: 0.5rem; padding: 0.5rem 1rem; border-bottom: 1px solid var(--dim); overflow-x: auto; }
  nav a { color: var(--dim); text-decoration: none; padding: 0.25rem 0.75rem; border: 1px solid transparent; }
  nav a:hover, nav a.active { color: var(--fg); border-color: var(--dim); border-radius: 2px; }
  main { padding: 1rem; max-width: 800px; margin: 0 auto; }
  .placeholder { color: var(--dim); padding: 2rem; text-align: center; border: 1px dashed var(--dim); border-radius: 4px; margin: 1rem 0; }
</style>
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
    <div id="habits" class="tab"><div class="placeholder">Habits — coming soon</div></div>
    <div id="tasks" class="tab" hidden><div class="placeholder">Tasks — coming soon</div></div>
    <div id="finance" class="tab" hidden><div class="placeholder">Finance — coming soon</div></div>
    <div id="calendar" class="tab" hidden><div class="placeholder">Calendar — coming soon</div></div>
    <div id="knowledge" class="tab" hidden><div class="placeholder">Knowledge — coming soon</div></div>
    <div id="chat" class="tab" hidden><div class="placeholder">Chat — coming soon</div></div>
  </main>
  <script>
    document.querySelectorAll('nav a').forEach(a => {
      a.addEventListener('click', e => {
        e.preventDefault();
        document.querySelectorAll('nav a').forEach(l => l.classList.remove('active'));
        a.classList.add('active');
        document.querySelectorAll('.tab').forEach(t => t.hidden = true);
        document.getElementById(a.dataset.tab).hidden = false;
      });
    });
    document.querySelector('nav a[data-tab="habits"]').classList.add('active');
    setInterval(() => { document.getElementById('time').textContent = new Date().toLocaleTimeString(); }, 1000);
  </script>
</body>
</html>`
