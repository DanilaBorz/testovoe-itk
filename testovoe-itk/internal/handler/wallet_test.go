package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	apperrors "https://github.com/DanilaBorz/testovoe-itk/internal/errors"
	"https://github.com/DanilaBorz/testovoe-itk/internal/handler"
	"https://github.com/DanilaBorz/testovoe-itk/internal/mocks"
	"https://github.com/DanilaBorz/testovoe-itk/internal/model"
)

func setupHandlerTest(t *testing.T) (*mocks.MockWalletService, *chi.Mux) {
	t.Helper()

	ctrl := gomock.NewController(t)
	mockSvc := mocks.NewMockWalletService(ctrl)
	logger, _ := zap.NewDevelopment()
	h := handler.NewWalletHandler(mockSvc, logger)

	r := chi.NewRouter()
	r.Post("/api/v1/wallet", h.HandleOperation)
	r.Get("/api/v1/wallets/{id}", h.HandleGetBalance)

	return mockSvc, r
}

func TestHandleOperation_Success_Deposit(t *testing.T) {
	mockSvc, router := setupHandlerTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationDeposit,
		Amount:        1000,
	}

	mockSvc.EXPECT().ProcessOperation(gomock.Any(), req).Return(int64(1000), nil)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.BalanceResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, walletID, resp.WalletID)
	assert.Equal(t, int64(1000), resp.Balance)
}

func TestHandleOperation_Success_Withdraw(t *testing.T) {
	mockSvc, router := setupHandlerTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationWithdraw,
		Amount:        500,
	}

	mockSvc.EXPECT().ProcessOperation(gomock.Any(), req).Return(int64(500), nil)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.BalanceResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, walletID, resp.WalletID)
	assert.Equal(t, int64(500), resp.Balance)
}

func TestHandleOperation_InvalidBody(t *testing.T) {
	_, router := setupHandlerTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader([]byte("invalid json")))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp model.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)
	assert.Contains(t, errResp.Message, "invalid request body")
}

func TestHandleOperation_MissingWalletID(t *testing.T) {
	_, router := setupHandlerTest(t)

	req := model.OperationRequest{
		WalletID:      uuid.Nil,
		OperationType: model.OperationDeposit,
		Amount:        1000,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleOperation_InvalidOperationType(t *testing.T) {
	_, router := setupHandlerTest(t)

	req := map[string]interface{}{
		"walletId":      uuid.New().String(),
		"operationType": "INVALID",
		"amount":        1000,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleOperation_ZeroAmount(t *testing.T) {
	_, router := setupHandlerTest(t)

	req := model.OperationRequest{
		WalletID:      uuid.New(),
		OperationType: model.OperationDeposit,
		Amount:        0,
	}

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleOperation_WalletNotFound(t *testing.T) {
	mockSvc, router := setupHandlerTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationWithdraw,
		Amount:        1000,
	}

	mockSvc.EXPECT().ProcessOperation(gomock.Any(), req).Return(int64(0), apperrors.ErrWalletNotFound)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleOperation_InsufficientFunds(t *testing.T) {
	mockSvc, router := setupHandlerTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationWithdraw,
		Amount:        1000,
	}

	mockSvc.EXPECT().ProcessOperation(gomock.Any(), req).Return(int64(0), apperrors.ErrInsufficientFunds)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleOperation_InternalError(t *testing.T) {
	mockSvc, router := setupHandlerTest(t)

	walletID := uuid.New()
	req := model.OperationRequest{
		WalletID:      walletID,
		OperationType: model.OperationDeposit,
		Amount:        1000,
	}

	mockSvc.EXPECT().ProcessOperation(gomock.Any(), req).Return(int64(0), errors.New("db connection failed"))

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallet", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandleOperation_WrongMethod(t *testing.T) {
	_, router := setupHandlerTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/wallet", nil)

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandleGetBalance_Success(t *testing.T) {
	mockSvc, router := setupHandlerTest(t)

	walletID := uuid.New()
	mockSvc.EXPECT().GetBalance(gomock.Any(), walletID).Return(int64(1500), nil)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+walletID.String(), nil)

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp model.BalanceResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, walletID, resp.WalletID)
	assert.Equal(t, int64(1500), resp.Balance)
}

func TestHandleGetBalance_NotFound(t *testing.T) {
	mockSvc, router := setupHandlerTest(t)

	walletID := uuid.New()
	mockSvc.EXPECT().GetBalance(gomock.Any(), walletID).Return(int64(0), apperrors.ErrWalletNotFound)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/"+walletID.String(), nil)

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleGetBalance_InvalidUUID(t *testing.T) {
	_, router := setupHandlerTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/wallets/not-a-uuid", nil)

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleGetBalance_WrongMethod(t *testing.T) {
	_, router := setupHandlerTest(t)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/wallets/"+uuid.New().String(), nil)

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}
