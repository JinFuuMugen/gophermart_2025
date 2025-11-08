package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
	"github.com/JinFuuMugen/gophermart-ya/internal/models"
	"github.com/JinFuuMugen/gophermart-ya/internal/storage"
	"github.com/golang-jwt/jwt/v5"
)

type UserHandler struct {
	DB        *storage.Database
	JWTSecret []byte
}

func (h *UserHandler) generateToken(login string) (string, error) {
	claims := jwt.MapClaims{
		"user": login,
		"exp":  time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(h.JWTSecret)
}

func (h *UserHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var u models.User
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if u.Login == "" || u.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}

	taken, err := h.DB.IsLoginTaken(u.Login)
	if err != nil {
		logger.Errorf("error checking login: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if taken {
		http.Error(w, "login already taken", http.StatusConflict)
		return
	}

	if err := h.DB.RegisterUser(u.Login, u.Password); err != nil {
		logger.Errorf("error registering user: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	token, err := h.generateToken(u.Login)
	if err != nil {
		logger.Errorf("failed to generate token: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   3600,
	})

	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds models.User
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if creds.Login == "" || creds.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}

	ok, err := h.DB.AuthenticateUser(creds.Login, creds.Password)
	if err != nil {
		logger.Errorf("error authenticating user: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "invalid login or password", http.StatusUnauthorized)
		return
	}

	token, err := h.generateToken(creds.Login)
	if err != nil {
		logger.Errorf("failed to generate token: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   3600,
	})

	w.WriteHeader(http.StatusOK)
}
func (h *UserHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil || cookie.Value == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(cookie.Value, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("invalid signing method")
			}
			return h.JWTSecret, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		login, _ := claims["user"].(string)
		if login == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "userLogin", login)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
