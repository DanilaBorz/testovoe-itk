package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	apperrors "github.com/DanilaBorz/testovoe-itk/internal/errors"
)

// WalletRepo реализует интерфейс service.WalletRepository.
type WalletRepo struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewWalletRepository создает новый WalletRepo.
func NewWalletRepository(pool *pgxpool.Pool, logger *zap.Logger) *WalletRepo {
	return &WalletRepo{
		pool:   pool,
		logger: logger,
	}
}

func (r *WalletRepo) GetBalance(ctx context.Context, id uuid.UUID) (int64, error) {
	query := `SELECT balance FROM wallets WHERE id = $1`

	var balance int64
	err := r.pool.QueryRow(ctx, query, id).Scan(&balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, apperrors.ErrWalletNotFound
		}
		r.logger.Error("failed to query wallet balance",
			zap.String("wallet_id", id.String()),
			zap.Error(err),
		)
		return 0, err
	}

	return balance, nil
}

func (r *WalletRepo) UpdateBalance(ctx context.Context, id uuid.UUID, amount int64) (newBalance int64, err error) {
	if amount >= 0 {
		return r.deposit(ctx, id, amount)
	}

	return r.withdraw(ctx, id, -amount)
}

func (r *WalletRepo) deposit(ctx context.Context, id uuid.UUID, amount int64) (int64, error) {
	query := `
		INSERT INTO wallets (id, balance)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE
		SET balance = wallets.balance + EXCLUDED.balance,
			updated_at = NOW()
		RETURNING balance
	`

	var newBalance int64
	if err := r.pool.QueryRow(ctx, query, id, amount).Scan(&newBalance); err != nil {
		r.logger.Error("failed to deposit wallet balance",
			zap.String("wallet_id", id.String()),
			zap.Int64("amount", amount),
			zap.Error(err),
		)
		return 0, err
	}

	r.logger.Debug("wallet balance deposited",
		zap.String("wallet_id", id.String()),
		zap.Int64("delta", amount),
		zap.Int64("new_balance", newBalance),
	)

	return newBalance, nil
}

func (r *WalletRepo) withdraw(ctx context.Context, id uuid.UUID, amount int64) (int64, error) {
	query := `
		UPDATE wallets
		SET balance = balance - $2,
			updated_at = NOW()
		WHERE id = $1 AND balance >= $2
		RETURNING balance
	`

	var newBalance int64
	err := r.pool.QueryRow(ctx, query, id, amount).Scan(&newBalance)
	if err == nil {
		r.logger.Debug("wallet balance withdrawn",
			zap.String("wallet_id", id.String()),
			zap.Int64("delta", -amount),
			zap.Int64("new_balance", newBalance),
		)

		return newBalance, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		r.logger.Error("failed to withdraw wallet balance",
			zap.String("wallet_id", id.String()),
			zap.Int64("amount", amount),
			zap.Error(err),
		)
		return 0, err
	}

	exists, err := r.walletExists(ctx, id)
	if err != nil {
		r.logger.Error("failed to check wallet existence after withdraw",
			zap.String("wallet_id", id.String()),
			zap.Error(err),
		)
		return 0, err
	}
	if !exists {
		return 0, apperrors.ErrWalletNotFound
	}

	return 0, apperrors.ErrInsufficientFunds
}

func (r *WalletRepo) walletExists(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM wallets WHERE id = $1)`

	var exists bool
	if err := r.pool.QueryRow(ctx, query, id).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}
