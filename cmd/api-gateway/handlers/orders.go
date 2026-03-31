package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/verno/gotradex/cmd/api-gateway/clients"
	"github.com/verno/gotradex/cmd/api-gateway/middleware"
)

type OrderHandler struct {
	orderClient clients.OrderServiceClient
}

func NewOrderHandler(orderClient clients.OrderServiceClient) *OrderHandler {
	return &OrderHandler{
		orderClient: orderClient,
	}
}

type PlaceOrderRequest struct {
	Symbol          string `json:"symbol" binding:"required"`
	Side            string `json:"side" binding:"required,oneof=BUY SELL"`
	Type            string `json:"type" binding:"required,oneof=LIMIT MARKET"`
	Price           string `json:"price"`
	Quantity        string `json:"quantity" binding:"required"`
	IdempotencyKey  string `json:"idempotency_key" binding:"required"`
}

func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	userID, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
			"code":  "UNAUTHORIZED",
		})
		return
	}

	var req PlaceOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	// Validate price is required for LIMIT orders
	if req.Type == "LIMIT" && req.Price == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Price is required for LIMIT orders",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	grpcReq := &clients.PlaceOrderRequest{
		UserID:         userID.(string),
		Symbol:         req.Symbol,
		Side:           req.Side,
		Type:           req.Type,
		Price:          req.Price,
		Quantity:       req.Quantity,
		IdempotencyKey: req.IdempotencyKey,
	}

	resp, err := h.orderClient.PlaceOrder(c.Request.Context(), grpcReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "ORDER_PLACEMENT_FAILED",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"order_id":   resp.OrderID,
		"status":     resp.Status,
		"symbol":     resp.Symbol,
		"side":       resp.Side,
		"type":       resp.Type,
		"price":      resp.Price,
		"quantity":   resp.Quantity,
		"filled_qty": resp.FilledQty,
	})
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Order ID is required",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	resp, err := h.orderClient.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Order not found",
			"code":  "ORDER_NOT_FOUND",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":   resp.OrderID,
		"user_id":    resp.UserID,
		"symbol":     resp.Symbol,
		"side":      resp.Side,
		"type":      resp.Type,
		"price":     resp.Price,
		"quantity":  resp.Quantity,
		"filled_qty": resp.FilledQty,
		"status":    resp.Status,
	})
}
