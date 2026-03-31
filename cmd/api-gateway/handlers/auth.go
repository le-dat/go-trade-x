package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/verno/gotradex/cmd/api-gateway/clients"
)

type AuthHandler struct {
	userClient clients.UserServiceClient
}

func NewAuthHandler(userClient clients.UserServiceClient) *AuthHandler {
	return &AuthHandler{
		userClient: userClient,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	resp, err := h.userClient.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
			"code":  "REGISTRATION_FAILED",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"user_id": resp.UserID,
		"email":   resp.Email,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"code":  "INVALID_REQUEST",
		})
		return
	}

	resp, err := h.userClient.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
			"code":  "INVALID_CREDENTIALS",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":  resp.Token,
		"user_id": resp.UserID,
		"email":   resp.Email,
	})
}
