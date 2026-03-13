package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/loks1k192/task-manager/internal/delivery/http/handler"
	"github.com/loks1k192/task-manager/internal/delivery/http/middleware"
	"github.com/loks1k192/task-manager/internal/metrics"
	"github.com/loks1k192/task-manager/internal/usecase"
)

func NewRouter(
	authUC *usecase.AuthUseCase,
	teamUC *usecase.TeamUseCase,
	taskUC *usecase.TaskUseCase,
) chi.Router {
	r := chi.NewRouter()

	r.Use(chimw.Recoverer)
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(metrics.Middleware)

	authH := handler.NewAuthHandler(authUC)
	teamH := handler.NewTeamHandler(teamUC)
	taskH := handler.NewTaskHandler(taskUC)

	rl := middleware.NewRateLimiter(100)

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(authUC))
			r.Use(rl.Middleware)

			r.Post("/teams", teamH.Create)
			r.Get("/teams", teamH.List)
			r.Post("/teams/{id}/invite", teamH.Invite)
			r.Get("/teams/stats", teamH.Stats)
			r.Get("/teams/top-creators", teamH.TopCreators)

			r.Post("/tasks", taskH.Create)
			r.Get("/tasks", taskH.List)
			r.Put("/tasks/{id}", taskH.Update)
			r.Get("/tasks/{id}/history", taskH.History)
			r.Get("/tasks/orphaned", taskH.Orphaned)

			r.Post("/tasks/{id}/comments", taskH.CreateComment)
			r.Get("/tasks/{id}/comments", taskH.ListComments)
		})
	})

	return r
}
