package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	return r
}

type registerReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}
type loginReq struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := h.svc.Register(r.Context(), req.Login, req.Password); err != nil {
		code := http.StatusInternalServerError
		switch err {
		case ErrPasswordTooShort:
			code = http.StatusBadRequest
		case ErrUserExists:
			code = http.StatusConflict
		}
		http.Error(w, err.Error(), code)
		return
	}

	token, err := h.svc.Authenticate(r.Context(), req.Login, req.Password)
	if err != nil {
		http.Error(w, "ошибка авторизации после регистрации", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token, err := h.svc.Authenticate(r.Context(), req.Login, req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	w.Header().Set("Authorization", "Bearer "+token)
}
