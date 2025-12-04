package accrual

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/JinFuuMugen/gophermart-ya/internal/logger"
	"github.com/JinFuuMugen/gophermart-ya/internal/storage"
)

func Worker(ctx context.Context, db *storage.Database, accrualAddr string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	client := &http.Client{Timeout: 5 * time.Second}

	for {
		select {
		case <-ctx.Done():

			logger.Infof("accrual worker stopped by context")
			return
		case <-ticker.C:
		}
		orders, err := db.GetPendingOrders()
		if err != nil {
			logger.Errorf("get pending orders: %v", err)
			continue
		}

		if len(orders) == 0 {
			continue
		}
		for _, order := range orders {
			select {
			case <-ctx.Done():
				logger.Infof("accrual worker stopped while processing orders")
				return
			default:
			}

			url := fmt.Sprintf("%s/api/orders/%s", accrualAddr, order.Number)

			resp, err := client.Get(url)
			if err != nil {
				logger.Errorf("accrual request failed for order %s: %v", order.Number, err)
				continue
			}

			if resp.StatusCode == http.StatusTooManyRequests {
				retryHeader := resp.Header.Get("Retry-After")
				resp.Body.Close()

				retrySeconds, err := strconv.Atoi(retryHeader)
				if err != nil {
					logger.Errorf("cannot parse Retry-After header %q: %v", retryHeader, err)
					continue
				}

				if retrySeconds > 0 {
					logger.Infof("accrual service returned 429, sleeping for %d seconds", retrySeconds)

					timer := time.NewTimer(time.Duration(retrySeconds) * time.Second)
					select {
					case <-timer.C:

					case <-ctx.Done():
						timer.Stop()
						logger.Infof("accrual worker stopped during 429 sleep")
						return
					}
				}

				break
			}

			if resp.StatusCode == http.StatusNoContent {
				resp.Body.Close()
				continue
			}

			if resp.StatusCode != http.StatusOK {
				logger.Errorf("unexpected status %d from accrual for order %s", resp.StatusCode, order.Number)
				resp.Body.Close()
				continue
			}

			var data struct {
				Order   string  `json:"order"`
				Status  string  `json:"status"`
				Accrual float64 `json:"accrual"`
			}

			if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
				logger.Errorf("decode accrual response for order %s: %v", order.Number, err)
				resp.Body.Close()
				continue
			}
			resp.Body.Close()

			if err := db.UpdateOrderAndBalance(data.Order, data.Status, data.Accrual); err != nil {
				logger.Errorf("update order %s: %v", data.Order, err)
			}
		}
	}
}
