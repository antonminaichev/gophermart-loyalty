// internal/order/handler.go
package order

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/antonminaichev/gophermart-loyalty/internal/middleware"
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
	r.Post("/orders", h.SubmitOrder)
	r.Get("/orders", h.ListOrders)
	return r
}

func (h *Handler) SubmitOrder(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	body, _ := io.ReadAll(r.Body)
	number := strings.TrimSpace(string(body))
	err := h.svc.SubmitOrder(r.Context(), userID, number)
	switch err {
	case ErrInvalidNumber:
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
	case ErrOrderAlreadyExists:
		w.WriteHeader(http.StatusOK)
	case ErrOrderAccepted:
		w.WriteHeader(http.StatusAccepted)
	case ErrOrderConflict:
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	fmt.Print(userID)
	orders, err := h.svc.ListOrders(r.Context(), userID)
	fmt.Print(orders)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
