package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrors "github.com/DanilaBorz/testovoe-itk/internal/errors"
	"github.com/DanilaBorz/testovoe-itk/internal/model"
	"github.com/DanilaBorz/testovoe-itk/internal/service"
)

// WalletHandler обрабатывает HTTP-запросы для операций с кошельком.
type WalletHandler struct {
	svc    service.WalletService
	logger *zap.Logger
}

// NewWalletHandler создает новый WalletHandler.
func NewWalletHandler(svc service.WalletService, logger *zap.Logger) *WalletHandler {
	return &WalletHandler{
		svc:    svc,
		logger: logger,
	}
}

// HandleOperation обрабатывает POST /api/v1/wallet.
func (h *WalletHandler) HandleOperation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req model.OperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("invalid request body", zap.Error(err))
		h.writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Проверка обязательных полей
	if req.WalletID == uuid.Nil {
		h.writeError(w, http.StatusBadRequest, "walletId is required")
		return
	}
	if req.OperationType != model.OperationDeposit && req.OperationType != model.OperationWithdraw {
		h.writeError(w, http.StatusBadRequest, "operationType must be DEPOSIT or WITHDRAW")
		return
	}
	if req.Amount <= 0 {
		h.writeError(w, http.StatusBadRequest, "amount must be positive")
		return
	}

	newBalance, err := h.svc.ProcessOperation(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, apperrors.ErrWalletNotFound):
			h.writeError(w, http.StatusNotFound, "wallet not found")
		case errors.Is(err, apperrors.ErrInsufficientFunds):
			h.writeError(w, http.StatusBadRequest, "insufficient funds")
		default:
			h.logger.Error("internal error processing operation",
				zap.String("wallet_id", req.WalletID.String()),
				zap.Error(err),
			)
			h.writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	resp := model.BalanceResponse{
		WalletID: req.WalletID,
		Balance:  newBalance,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

// HandleGetBalance обрабатывает GET /api/v1/wallets/{id}.
func (h *WalletHandler) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.logger.Warn("invalid wallet id format", zap.String("id", idStr))
		h.writeError(w, http.StatusBadRequest, "invalid wallet id format")
		return
	}

	balance, err := h.svc.GetBalance(r.Context(), id)
	if err != nil {
		if errors.Is(err, apperrors.ErrWalletNotFound) {
			h.writeError(w, http.StatusNotFound, "wallet not found")
			return
		}
		h.logger.Error("internal error getting balance",
			zap.String("wallet_id", id.String()),
			zap.Error(err),
		)
		h.writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	resp := model.BalanceResponse{
		WalletID: id,
		Balance:  balance,
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *WalletHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
	}
}

func (h *WalletHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, model.ErrorResponse{Message: message})
}
