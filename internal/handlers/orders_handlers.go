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
	DB *storage.Database
}

func isValidLuhn(num string) bool {
	var sum int
	alt := false
	for i := len(num) - 1; i >= 0; i-- {
		d := int(num[i] - '0')
		if d < 0 || d > 9 {
			return false
		}
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		alt = !alt
	}
	return sum%10 == 0
}

func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	login, ok := userFromContext(r.Context())
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

	if !isValidLuhn(orderNum) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	code, err := h.DB.CheckOrderOwner(orderNum, login)
	if err != nil {
		logger.Errorf("error checking order: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	switch code {
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
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	login, ok := userFromContext(r.Context())
	if !ok || login == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.DB.GetOrders(login)
	if err != nil {
		logger.Errorf("error getting orders: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_ = json.NewEncoder(w).Encode(orders)
}
