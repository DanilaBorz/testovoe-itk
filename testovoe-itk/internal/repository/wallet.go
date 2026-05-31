package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	apperrors "https://github.com/DanilaBorz/testovoe-itk/internal/errors"
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("failed to begin transaction",
			zap.String("wallet_id", id.String()),
			zap.Error(err),
		)
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx) //nolint:errcheck — транзакция может быть уже закрыта
		}
	}()

	// Попытка заблокировать существующую запись кошелька
	var currentBalance int64
	query := `SELECT balance FROM wallets WHERE id = $1 FOR UPDATE`
	err = tx.QueryRow(ctx, query, id).Scan(&currentBalance)

	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			r.logger.Error("failed to lock wallet row",
				zap.String("wallet_id", id.String()),
				zap.Error(err),
			)
			return 0, err
		}

		// Кошелек не существует — создаем его с указанной суммой (только для DEPOSIT)
		if amount < 0 {
			return 0, apperrors.ErrWalletNotFound
		}

		insertQuery := `INSERT INTO wallets (id, balance) VALUES ($1, $2)`
		_, err = tx.Exec(ctx, insertQuery, id, amount)
		if err != nil {
			r.logger.Error("failed to create wallet",
				zap.String("wallet_id", id.String()),
				zap.Error(err),
			)
			return 0, err
		}

		err = tx.Commit(ctx)
		if err != nil {
			r.logger.Error("failed to commit transaction after wallet creation",
				zap.String("wallet_id", id.String()),
				zap.Error(err),
			)
			return 0, err
		}

		r.logger.Info("wallet created",
			zap.String("wallet_id", id.String()),
			zap.Int64("initial_balance", amount),
		)

		return amount, nil
	}

	// Кошелек существует — обновляем баланс
	newBalance = currentBalance + amount
	if newBalance < 0 {
		return 0, apperrors.ErrInsufficientFunds
	}

	updateQuery := `UPDATE wallets SET balance = $1, updated_at = NOW() WHERE id = $2`
	_, err = tx.Exec(ctx, updateQuery, newBalance, id)
	if err != nil {
		r.logger.Error("failed to update wallet balance",
			zap.String("wallet_id", id.String()),
			zap.Int64("new_balance", newBalance),
			zap.Error(err),
		)
		return 0, err
	}

	err = tx.Commit(ctx)
	if err != nil {
		r.logger.Error("failed to commit transaction after balance update",
			zap.String("wallet_id", id.String()),
			zap.Error(err),
		)
		return 0, err
	}

	r.logger.Info("wallet balance updated",
		zap.String("wallet_id", id.String()),
		zap.Int64("delta", amount),
		zap.Int64("new_balance", newBalance),
	)

	return newBalance, nil
}