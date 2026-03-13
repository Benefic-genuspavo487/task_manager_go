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

type TaskHandler struct {
	taskUC *usecase.TaskUseCase
}

func NewTaskHandler(taskUC *usecase.TaskUseCase) *TaskHandler {
	return &TaskHandler{taskUC: taskUC}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateTaskInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := validator.Validate(input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	userID := middleware.GetUserID(r.Context())
	task, err := h.taskUC.Create(r.Context(), input, userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := domain.TaskFilter{Limit: 20}

	if v := r.URL.Query().Get("team_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid team_id"})
			return
		}
		filter.TeamID = &id
	}
	if v := r.URL.Query().Get("status"); v != "" {
		s := domain.TaskStatus(v)
		filter.Status = &s
	}
	if v := r.URL.Query().Get("assignee_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid assignee_id"})
			return
		}
		filter.AssigneeID = &id
	}
	if v := r.URL.Query().Get("cursor"); v != "" {
		c, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid cursor"})
			return
		}
		filter.Cursor = &c
	}
	if v := r.URL.Query().Get("limit"); v != "" {
		l, err := strconv.Atoi(v)
		if err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	tasks, err := h.taskUC.List(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	if tasks == nil {
		tasks = make([]domain.Task, 0)
	}

	type paginatedResponse struct {
		Data       []domain.Task `json:"data"`
		NextCursor *int64        `json:"next_cursor,omitempty"`
	}
	resp := paginatedResponse{Data: tasks}
	if len(tasks) == filter.Limit {
		lastID := tasks[len(tasks)-1].ID
		resp.NextCursor = &lastID
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id"})
		return
	}

	var input usecase.UpdateTaskInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := validator.Validate(input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	userID := middleware.GetUserID(r.Context())
	task, err := h.taskUC.Update(r.Context(), taskID, input, userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, task)
}

func (h *TaskHandler) History(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id"})
		return
	}

	history, err := h.taskUC.GetHistory(r.Context(), taskID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, history)
}

func (h *TaskHandler) Orphaned(w http.ResponseWriter, r *http.Request) {
	orphaned, err := h.taskUC.FindOrphaned(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, orphaned)
}

func (h *TaskHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id"})
		return
	}

	var input usecase.CreateCommentInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}
	if err := validator.Validate(input); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	userID := middleware.GetUserID(r.Context())
	comment, err := h.taskUC.CreateComment(r.Context(), taskID, input, userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (h *TaskHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid task id"})
		return
	}

	comments, err := h.taskUC.ListComments(r.Context(), taskID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, comments)
}
