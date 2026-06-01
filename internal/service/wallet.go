package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/DanilaBorz/testovoe-itk/internal/model"
)

// WalletRepository определяет интерфейс для доступа к данным кошелька.
type WalletRepository interface {
	// GetBalance возвращает баланс кошелька по ID.
	GetBalance(ctx context.Context, id uuid.UUID) (int64, error)
	// UpdateBalance атомарно обновляет баланс кошелька.
	// Если кошелек не существует, он будет создан с указанной суммой.
	// Возвращает новый баланс.
	UpdateBalance(ctx context.Context, id uuid.UUID, amount int64) (int64, error)
}

// WalletService определяет интерфейс бизнес-логики кошелька.
type WalletService interface {
	// ProcessOperation обрабатывает операцию пополнения или снятия.
	ProcessOperation(ctx context.Context, req model.OperationRequest) (int64, error)
	// GetBalance возвращает текущий баланс кошелька.
	GetBalance(ctx context.Context, id uuid.UUID) (int64, error)
}

type walletService struct {
	repo   WalletRepository
	logger *zap.Logger
}

// NewWalletService создает новый WalletService.
func NewWalletService(repo WalletRepository, logger *zap.Logger) WalletService {
	return &walletService{
		repo:   repo,
		logger: logger,
	}
}

func (s *walletService) ProcessOperation(ctx context.Context, req model.OperationRequest) (int64, error) {
	var amount int64

	switch req.OperationType {
	case model.OperationDeposit:
		amount = req.Amount
	case model.OperationWithdraw:
		amount = -req.Amount
	default:
		return 0, errors.New("invalid operation type")
	}

	newBalance, err := s.repo.UpdateBalance(ctx, req.WalletID, amount)
	if err != nil {
		s.logger.Warn("failed to process operation",
			zap.String("wallet_id", req.WalletID.String()),
			zap.String("operation", string(req.OperationType)),
			zap.Int64("amount", req.Amount),
			zap.Error(err),
		)
		return 0, err
	}

	s.logger.Debug("operation processed",
		zap.String("wallet_id", req.WalletID.String()),
		zap.String("operation", string(req.OperationType)),
		zap.Int64("amount", req.Amount),
		zap.Int64("new_balance", newBalance),
	)

	return newBalance, nil
}

func (s *walletService) GetBalance(ctx context.Context, id uuid.UUID) (int64, error) {
	balance, err := s.repo.GetBalance(ctx, id)
	if err != nil {
		s.logger.Warn("failed to get balance",
			zap.String("wallet_id", id.String()),
			zap.Error(err),
		)
		return 0, err
	}

	return balance, nil
}
