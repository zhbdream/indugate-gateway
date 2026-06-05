package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/middleware"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/model"
	"github.com/indugate/gateway/internal/service"
)

type UserHandler struct {
	auth *service.AuthService
	perm *service.DevicePermissionService
}

func NewUserHandler(auth *service.AuthService, perm *service.DevicePermissionService) *UserHandler {
	return &UserHandler{auth: auth, perm: perm}
}

func (h *UserHandler) List(c *gin.Context) {
	if !middleware.RequireAdmin(c) {
		return
	}
	users, err := h.auth.ListUsers(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	items := make([]gin.H, 0, len(users))
	for _, u := range users {
		items = append(items, userResponse(u))
	}
	response.OK(c, items)
}

type createUserRequest struct {
	Username string         `json:"username" binding:"required"`
	Password string         `json:"password" binding:"required"`
	Role     model.UserRole `json:"role"`
}

func (h *UserHandler) Create(c *gin.Context) {
	if !middleware.RequireAdmin(c) {
		return
	}
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.auth.CreateUser(c.Request.Context(), req.Username, req.Password, req.Role)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Created(c, userResponse(*user))
}

type updateUserRequest struct {
	Role model.UserRole `json:"role" binding:"required"`
}

func (h *UserHandler) Update(c *gin.Context) {
	if !middleware.RequireAdmin(c) {
		return
	}
	id, err := parseUserID(c)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	user, err := h.auth.UpdateUser(c.Request.Context(), id, req.Role)
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, userResponse(*user))
}

type changePasswordRequest struct {
	Password string `json:"password" binding:"required"`
}

func (h *UserHandler) ChangePassword(c *gin.Context) {
	if !middleware.RequireAdmin(c) {
		return
	}
	id, err := parseUserID(c)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.auth.ChangePassword(c.Request.Context(), id, req.Password); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) Delete(c *gin.Context) {
	if !middleware.RequireAdmin(c) {
		return
	}
	id, err := parseUserID(c)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	if err := h.auth.DeleteUser(c.Request.Context(), id); err != nil {
		if errors.Is(err, service.ErrLastAdmin) {
			response.Fail(c, 409, 409, err.Error())
			return
		}
		response.NotFound(c, err.Error())
		return
	}
	response.OK(c, nil)
}

func (h *UserHandler) ListDevices(c *gin.Context) {
	if !middleware.RequireAdmin(c) {
		return
	}
	id, err := parseUserID(c)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	deviceIDs, err := h.perm.GetUserDeviceIDs(c.Request.Context(), id)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, gin.H{"device_ids": deviceIDs})
}

type setUserDevicesRequest struct {
	DeviceIDs []uint `json:"device_ids"`
}

func (h *UserHandler) SetDevices(c *gin.Context) {
	if !middleware.RequireAdmin(c) {
		return
	}
	id, err := parseUserID(c)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req setUserDevicesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if _, err := h.auth.GetUser(c.Request.Context(), id); err != nil {
		response.NotFound(c, err.Error())
		return
	}
	if err := h.perm.SetUserDevices(c.Request.Context(), id, req.DeviceIDs); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"device_ids": req.DeviceIDs})
}

func parseUserID(c *gin.Context) (uint, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	return uint(id), err
}

func userResponse(u model.User) gin.H {
	return gin.H{
		"id":         u.ID,
		"username":   u.Username,
		"role":       u.Role,
		"created_at": u.CreatedAt,
		"updated_at": u.UpdatedAt,
	}
}
