package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
	"github.com/JinFuuMugen/gophermart-ya/internal/storage"
)

type OrderHandler struct {
	DB        *storage.Database
	JWTSecret []byte
}

func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	login, ok := r.Context().Value("userLogin").(string)
	if !ok || login == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	orderNum := strings.TrimSpace(string(body))
	if orderNum == "" {
		http.Error(w, "empty order number", http.StatusBadRequest)
		return
	}

	for _, c := range orderNum {
		if c < '0' || c > '9' {
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
			return
		}
	}

	statusCode, err := h.DB.CheckOrderOwner(orderNum, login)
	if err != nil {
		logger.Errorf("error checking order: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	switch statusCode {
	case 200:
		w.WriteHeader(http.StatusOK)
	case 409:
		http.Error(w, "order belongs to another user", http.StatusConflict)
	case 202:
		if err := h.DB.StoreOrder(orderNum, login); err != nil {
			logger.Errorf("error storing order: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "unexpected status", http.StatusInternalServerError)
	}
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	login, ok := r.Context().Value("userLogin").(string)
	if !ok || login == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.DB.GetOrders(login)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	json.NewEncoder(w).Encode(orders)
}
