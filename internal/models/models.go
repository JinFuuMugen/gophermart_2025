package models

import "time"

//
// Пользователь
//

// User — структура для входных данных при регистрации и логине.
// Используется в JSON-запросах /api/user/register и /api/user/login.
type User struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

//
// Заказы
//

// OrderStatus — допустимые статусы заказов согласно ТЗ.
type OrderStatus string

const (
	OrderStatusNew        OrderStatus = "NEW"        // заказ загружен, но не обработан
	OrderStatusProcessing OrderStatus = "PROCESSING" // заказ в обработке
	OrderStatusInvalid    OrderStatus = "INVALID"    // система расчёта отказала
	OrderStatusProcessed  OrderStatus = "PROCESSED"  // расчёт завершён, баллы начислены
)

// Order — сущность заказа в системе лояльности.
// Используется как в БД, так и для отдачи через API /api/user/orders.
type Order struct {
	Number     string      `json:"number"`            // номер заказа
	Status     OrderStatus `json:"status"`            // статус обработки
	Accrual    *float64    `json:"accrual,omitempty"` // начисленные баллы, опционально
	UploadedAt time.Time   `json:"uploaded_at"`       // время загрузки, RFC3339
}

//
// Баланс пользователя
//

// Balance — текущий баланс и сумма списаний.
// Используется в ответе /api/user/balance.
type Balance struct {
	Current   float64 `json:"current"`   // текущий баланс
	Withdrawn float64 `json:"withdrawn"` // общая сумма списаний
}

//
// Списание (вывод средств)
//

// Withdrawal — запись о списании средств.
// Используется в /api/user/withdrawals.
type Withdrawal struct {
	Order       string    `json:"order"`        // номер заказа, по которому произошло списание
	Sum         float64   `json:"sum"`          // сумма списания
	ProcessedAt time.Time `json:"processed_at"` // время списания, RFC3339
}

// WithdrawRequest — структура запроса на списание.
// Используется в /api/user/balance/withdraw.
type WithdrawRequest struct {
	Order string  `json:"order"` // номер заказа
	Sum   float64 `json:"sum"`   // сумма баллов к списанию
}
