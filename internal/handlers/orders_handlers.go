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

func isValidLuhn(number string) bool {
	var sum int
	var alt bool

	for i := len(number) - 1; i >= 0; i-- {
		n := int(number[i] - '0')
		if n < 0 || n > 9 {
			return false
		}
		if alt {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alt = !alt
	}
	return sum%10 == 0
}

func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	loginCookie, err := r.Cookie("auth_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	login := extractLoginFromToken(loginCookie.Value)
	if login == "" {
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

	// Проверяем алгоритмом Луна
	if !isValidLuhn(orderNum) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
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
		return
	case 409:
		http.Error(w, "order belongs to another user", http.StatusConflict)
		return
	case 202:
		if err := h.DB.StoreOrder(orderNum, login); err != nil {
			logger.Errorf("error storing order: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		return
	default:
		http.Error(w, "unexpected status", http.StatusInternalServerError)
	}
}

func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	loginCookie, err := r.Cookie("auth_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	login := extractLoginFromToken(loginCookie.Value)
	if login == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	orders, err := h.DB.GetOrders(login)
	if err != nil {
		logger.Errorf("error getting orders: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

func extractLoginFromToken(token string) string {
	return token // TODO:claims["user"]
}
