package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
	"github.com/JinFuuMugen/gophermart-ya/internal/storage"
	"github.com/golang-jwt/jwt/v5"
)

type OrderHandler struct {
	DB        *storage.Database
	JWTSecret []byte
}

func (h *OrderHandler) extractLoginFromToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return h.JWTSecret, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if user, ok := claims["user"].(string); ok {
			return user, nil
		}
	}

	return "", fmt.Errorf("no user in token")
}

func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	login, err := h.extractLoginFromToken(cookie.Value)
	if err != nil {
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

	// Заказ должен состоять только из цифр
	for _, c := range orderNum {
		if c < '0' || c > '9' {
			http.Error(w, "invalid order number format", http.StatusUnprocessableEntity)
			return
		}
	}

	// Проверяем существование заказа
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
	cookie, err := r.Cookie("auth_token")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	login, err := h.extractLoginFromToken(cookie.Value)
	if err != nil {
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

	json.NewEncoder(w).Encode(orders)
}
