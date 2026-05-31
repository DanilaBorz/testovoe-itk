package apperrors

import "errors"

var (
	// ErrWalletNotFound возвращается, когда кошелек не найден.
	ErrWalletNotFound = errors.New("wallet not found")
	// ErrInsufficientFunds возвращается, когда на кошельке недостаточно средств.
	ErrInsufficientFunds = errors.New("insufficient funds")
)