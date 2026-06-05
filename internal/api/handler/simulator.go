package handler

import (
	"io"

	"github.com/gin-gonic/gin"
	"github.com/indugate/gateway/internal/api/response"
	"github.com/indugate/gateway/internal/service"
)

type SimulatorHandler struct {
	svc *service.SimulatorService
}

func NewSimulatorHandler(svc *service.SimulatorService) *SimulatorHandler {
	return &SimulatorHandler{svc: svc}
}

func (h *SimulatorHandler) List(c *gin.Context) {
	response.OK(c, h.svc.List())
}

func (h *SimulatorHandler) Start(c *gin.Context) {
	simType := c.Param("type")
	result, err := h.svc.Start(simType)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *SimulatorHandler) Stop(c *gin.Context) {
	simType := c.Param("type")
	result, err := h.svc.Stop(simType)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, result)
}

func (h *SimulatorHandler) UpdateConfig(c *gin.Context) {
	simType := c.Param("type")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "failed to read request body")
		return
	}

	result, err := h.svc.UpdateConfig(simType, string(body))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.OK(c, result)
}
