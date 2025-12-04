package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
	"github.com/JinFuuMugen/gophermart-ya/internal/models"
	"github.com/JinFuuMugen/gophermart-ya/internal/storage"
)

type BalanceHandler struct {
	DB *storage.Database
}

func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	login, ok := userFromContext(r.Context())
	if !ok || login == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	current, withdrawn, err := h.DB.GetBalance(login)
	if err != nil {
		logger.Errorf("cannot get balance: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := models.BalanceResponse{
		Current:   current,
		Withdrawn: withdrawn,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *BalanceHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	login, ok := userFromContext(r.Context())
	if !ok || login == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req models.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if req.Order == "" || req.Sum <= 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if !isValidLuhn(req.Order) {
		http.Error(w, "invalid order number", http.StatusUnprocessableEntity)
		return
	}

	err := h.DB.Withdraw(login, req.Order, req.Sum)
	if err != nil {
		if err.Error() == "insufficient funds" {
			http.Error(w, "insufficient funds", http.StatusPaymentRequired)
			return
		}
		logger.Errorf("cannot withdraw: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	login, ok := userFromContext(r.Context())
	if !ok || login == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	withdrawals, err := h.DB.GetWithdrawals(login)
	if err != nil {
		logger.Errorf("cannot get withdrawals: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(withdrawals)
}
