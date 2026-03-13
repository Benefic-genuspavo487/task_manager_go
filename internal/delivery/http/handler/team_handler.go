package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/loks1k192/task-manager/internal/delivery/http/middleware"
	"github.com/loks1k192/task-manager/internal/domain"
	"github.com/loks1k192/task-manager/internal/usecase"
	"github.com/loks1k192/task-manager/pkg/validator"
)

type TeamHandler struct {
	teamUC *usecase.TeamUseCase
}

func NewTeamHandler(teamUC *usecase.TeamUseCase) *TeamHandler {
	return &TeamHandler{teamUC: teamUC}
}

func (h *TeamHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateTeamInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := validator.Validate(input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	userID := middleware.GetUserID(r.Context())
	team, err := h.teamUC.Create(r.Context(), input, userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, team)
}

func (h *TeamHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	teams, err := h.teamUC.ListByUser(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	if teams == nil {
		teams = make([]domain.Team, 0)
	}
	writeJSON(w, http.StatusOK, teams)
}

func (h *TeamHandler) Invite(w http.ResponseWriter, r *http.Request) {
	teamID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid team id"})
		return
	}

	var input usecase.InviteInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := validator.Validate(input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.teamUC.Invite(r.Context(), teamID, input, userID); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user invited"})
}

func (h *TeamHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.teamUC.GetStats(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *TeamHandler) TopCreators(w http.ResponseWriter, r *http.Request) {
	yearStr := r.URL.Query().Get("year")
	monthStr := r.URL.Query().Get("month")

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid year"})
		return
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid month"})
		return
	}

	top, err := h.teamUC.GetTopCreators(r.Context(), year, month)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, top)
}
