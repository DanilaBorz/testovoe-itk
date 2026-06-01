package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	apperrors "github.com/DanilaBorz/testovoe-itk/internal/errors"
	"github.com/DanilaBorz/testovoe-itk/internal/mocks"
	"github.com/DanilaBorz/testovoe-itk/internal/model"
	"github.com/DanilaBorz/testovoe-itk/internal/service"
)

func setupServiceTest(t *testing.T) (*mocks.MockWalletRepository, service.WalletService) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockRepo := mocks.NewMockWalletRepository(ctrl)
	logger, _ := zap.NewDevelopment()
	svc := service.NewWalletService(mockRepo, logger)

	return mockRepo, svc
}

func TestProcessOperation_Deposit(t *testing.T) {
	mockRepo, svc := setupServiceTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationDeposit,
		Amount:        1000,
	}

	mockRepo.EXPECT().UpdateBalance(gomock.Any(), walletID, int64(1000)).Return(int64(1000), nil)

	newBalance, err := svc.ProcessOperation(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(1000), newBalance)
}

func TestProcessOperation_Withdraw(t *testing.T) {
	mockRepo, svc := setupServiceTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationWithdraw,
		Amount:        500,
	}

	mockRepo.EXPECT().UpdateBalance(gomock.Any(), walletID, int64(-500)).Return(int64(500), nil)

	newBalance, err := svc.ProcessOperation(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, int64(500), newBalance)
}

func TestProcessOperation_InvalidType(t *testing.T) {
	_, svc := setupServiceTest(t)

	req := model.OperationRequest{
		WalletID:      uuid.New(),
		OperationType: "INVALID",
		Amount:        1000,
	}

	_, err := svc.ProcessOperation(context.Background(), req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid operation type")
}

func TestProcessOperation_WalletNotFound(t *testing.T) {
	mockRepo, svc := setupServiceTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationWithdraw,
		Amount:        1000,
	}

	mockRepo.EXPECT().UpdateBalance(gomock.Any(), walletID, int64(-1000)).Return(int64(0), apperrors.ErrWalletNotFound)

	_, err := svc.ProcessOperation(context.Background(), req)

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrWalletNotFound))
}

func TestProcessOperation_InsufficientFunds(t *testing.T) {
	mockRepo, svc := setupServiceTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationWithdraw,
		Amount:        999999,
	}

	mockRepo.EXPECT().UpdateBalance(gomock.Any(), walletID, int64(-999999)).Return(int64(0), apperrors.ErrInsufficientFunds)

	_, err := svc.ProcessOperation(context.Background(), req)

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrInsufficientFunds))
}

func TestGetBalance_Success(t *testing.T) {
	mockRepo, svc := setupServiceTest(t)

	walletID := uuid.New()
	mockRepo.EXPECT().GetBalance(gomock.Any(), walletID).Return(int64(5000), nil)

	balance, err := svc.GetBalance(context.Background(), walletID)

	require.NoError(t, err)
	assert.Equal(t, int64(5000), balance)
}

func TestGetBalance_NotFound(t *testing.T) {
	mockRepo, svc := setupServiceTest(t)

	walletID := uuid.New()
	mockRepo.EXPECT().GetBalance(gomock.Any(), walletID).Return(int64(0), apperrors.ErrWalletNotFound)

	_, err := svc.GetBalance(context.Background(), walletID)

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperrors.ErrWalletNotFound))
}

func TestGetBalance_RepoError(t *testing.T) {
	mockRepo, svc := setupServiceTest(t)

	walletID := uuid.New()
	mockRepo.EXPECT().GetBalance(gomock.Any(), walletID).Return(int64(0), errors.New("db error"))

	_, err := svc.GetBalance(context.Background(), walletID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}
