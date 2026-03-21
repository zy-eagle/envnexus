package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/service/enrollment"
)

type TokenHandler struct {
	enrollService *enrollment.Service
}

func NewTokenHandler(enrollService *enrollment.Service) *TokenHandler {
	return &TokenHandler{
		enrollService: enrollService,
	}
}

func (h *TokenHandler) RegisterRoutes(router *gin.RouterGroup) {
	tokens := router.Group("/tenants/:tenantId/tokens")
	{
		tokens.POST("", h.CreateToken)
	}
}

func (h *TokenHandler) CreateToken(c *gin.Context) {
	tenantID := c.Param("tenantId")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenantId is required"})
		return
	}

	var req dto.CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.enrollService.CreateToken(c.Request.Context(), tenantID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}
