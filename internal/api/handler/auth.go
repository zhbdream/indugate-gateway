package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/service"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Config(c *gin.Context) {
	response.OK(c, gin.H{
		"auth_enabled":       h.auth.Enabled(),
		"jwt_enabled":        h.auth.JWTEnabled(),
		"device_acl_enabled": h.auth.DeviceACLEnabled(),
	})
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	token, user, err := h.auth.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Fail(c, 401, 401, err.Error())
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		response.Fail(c, 401, 401, "unauthorized")
		return
	}
	id, ok := userID.(uint)
	if !ok {
		response.Fail(c, 401, 401, "unauthorized")
		return
	}
	user, err := h.auth.GetUser(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"role":     user.Role,
	})
}
