//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"

	apperrors "github.com/DanilaBorz/testovoe-itk/internal/errors"
	"github.com/DanilaBorz/testovoe-itk/internal/repository"
)

type WalletRepositoryTestSuite struct {
	suite.Suite
	pool   *pgxpool.Pool
	repo   *repository.WalletRepo
	logger *zap.Logger
}

func (s *WalletRepositoryTestSuite) SetupSuite() {
	dsn := "postgres://wallet_test:wallet_test_pass@localhost:5433/wallet_test_db?sslmode=disable"

	pool, err := pgxpool.New(context.Background(), dsn)
	require.NoError(s.T(), err)

	err = pool.Ping(context.Background())
	require.NoError(s.T(), err)

	logger, _ := zap.NewDevelopment()

	s.pool = pool
	s.logger = logger
	s.repo = repository.NewWalletRepository(pool, logger)
}

func (s *WalletRepositoryTestSuite) TearDownSuite() {
	s.pool.Close()
}

func (s *WalletRepositoryTestSuite) SetupTest() {
	// Очистка таблицы перед каждым тестом
	_, err := s.pool.Exec(context.Background(), "DELETE FROM wallets")
	require.NoError(s.T(), err)
}

func (s *WalletRepositoryTestSuite) TestCreateAndGetBalance() {
	walletID := uuid.New()

	// Создание кошелька через пополнение
	balance, err := s.repo.UpdateBalance(context.Background(), walletID, 1000)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1000), balance)

	// Получение баланса
	gotBalance, err := s.repo.GetBalance(context.Background(), walletID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1000), gotBalance)
}

func (s *WalletRepositoryTestSuite) TestDeposit() {
	walletID := uuid.New()

	// Создание кошелька
	_, err := s.repo.UpdateBalance(context.Background(), walletID, 1000)
	require.NoError(s.T(), err)

	// Пополнение
	balance, err := s.repo.UpdateBalance(context.Background(), walletID, 500)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1500), balance)
}

func (s *WalletRepositoryTestSuite) TestWithdraw() {
	walletID := uuid.New()

	// Создание кошелька
	_, err := s.repo.UpdateBalance(context.Background(), walletID, 1000)
	require.NoError(s.T(), err)

	// Снятие
	balance, err := s.repo.UpdateBalance(context.Background(), walletID, -500)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(500), balance)
}

func (s *WalletRepositoryTestSuite) TestWithdrawInsufficientFunds() {
	walletID := uuid.New()

	// Создание кошелька
	_, err := s.repo.UpdateBalance(context.Background(), walletID, 100)
	require.NoError(s.T(), err)

	// Попытка снять больше, чем есть на балансе
	_, err = s.repo.UpdateBalance(context.Background(), walletID, -200)
	require.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, apperrors.ErrInsufficientFunds)
}

func (s *WalletRepositoryTestSuite) TestWithdrawFromNonExistentWallet() {
	walletID := uuid.New()

	// Попытка снять с несуществующего кошелька
	_, err := s.repo.UpdateBalance(context.Background(), walletID, -100)
	require.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, apperrors.ErrWalletNotFound)
}

func (s *WalletRepositoryTestSuite) TestGetBalanceNonExistent() {
	walletID := uuid.New()

	_, err := s.repo.GetBalance(context.Background(), walletID)
	require.Error(s.T(), err)
	assert.ErrorIs(s.T(), err, apperrors.ErrWalletNotFound)
}

func (s *WalletRepositoryTestSuite) TestConcurrentOperations() {
	walletID := uuid.New()

	// Создание кошелька
	_, err := s.repo.UpdateBalance(context.Background(), walletID, 0)
	require.NoError(s.T(), err)

	// Запуск 100 конкурентных операций
	const ops = 100
	errs := make(chan error, ops)

	for i := 0; i < ops; i++ {
		go func() {
			_, err := s.repo.UpdateBalance(context.Background(), walletID, 10)
			errs <- err
		}()
	}

	for i := 0; i < ops; i++ {
		err := <-errs
		require.NoError(s.T(), err)
	}

	// Итоговый баланс должен быть 100 * 10 = 1000
	balance, err := s.repo.GetBalance(context.Background(), walletID)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), int64(1000), balance)
}

func TestWalletRepositorySuite(t *testing.T) {
	suite.Run(t, new(WalletRepositoryTestSuite))
}
