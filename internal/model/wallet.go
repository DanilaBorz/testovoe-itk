package model

import (
	"time"

	"github.com/google/uuid"
)

// OperationType представляет тип операции с кошельком.
type OperationType string

const (
	OperationDeposit  OperationType = "DEPOSIT"
	OperationWithdraw OperationType = "WITHDRAW"
)

// Wallet представляет сущность кошелька.
type Wallet struct {
	ID        uuid.UUID `json:"id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OperationRequest представляет входящий запрос POST /api/v1/wallet.
type OperationRequest struct {
	WalletID      uuid.UUID     `json:"walletId" validate:"required"`
	OperationType OperationType `json:"operationType" validate:"required,oneof=DEPOSIT WITHDRAW"`
	Amount        int64         `json:"amount" validate:"required,gt=0"`
}

// BalanceResponse представляет ответ GET /api/v1/wallets/{id}.
type BalanceResponse struct {
	WalletID uuid.UUID `json:"walletId"`
	Balance  int64     `json:"balance"`
}

// ErrorResponse представляет ответ с ошибкой.
type ErrorResponse struct {
	Message string `json:"message"`
}