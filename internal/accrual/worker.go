package accrual

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
	"github.com/JinFuuMugen/gophermart-ya/internal/storage"
)

func Worker(db *storage.Database, accrualAddr string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	client := &http.Client{Timeout: 5 * time.Second}

	for range ticker.C {
		orders, err := db.GetPendingOrders()
		if err != nil {
			logger.Errorf("get pending orders: %v", err)
			continue
		}

		for _, order := range orders {
			url := fmt.Sprintf("%s/api/orders/%s", accrualAddr, order.Number)

			resp, err := client.Get(url)
			if err != nil {
				logger.Errorf("accrual request failed: %v", err)
				continue
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				retry, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
				resp.Body.Close()
				if retry > 0 {
					time.Sleep(time.Duration(retry) * time.Second)
				}
				continue
			}

			if resp.StatusCode == http.StatusNoContent {
				resp.Body.Close()
				continue
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				logger.Errorf("unexpected status %d from accrual", resp.StatusCode)
				continue
			}

			var data struct {
				Order   string  `json:"order"`
				Status  string  `json:"status"`
				Accrual float64 `json:"accrual"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				resp.Body.Close()
				logger.Errorf("decode accrual response: %v", err)
				continue
			}
			resp.Body.Close()

			if err := db.UpdateOrderAndBalance(data.Order, data.Status, data.Accrual); err != nil {
				logger.Errorf("update order %s: %v", data.Order, err)
			}
		}
	}
}
