package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *LocalServer) RegisterFileRoutes(api *gin.RouterGroup) {
	files := api.Group("/files")
	{
		files.POST("/browse", s.handleFilesBrowse)
		files.POST("/preview", s.handleFilesPreview)
		files.POST("/download", s.handleFilesDownload)
	}
}

type filesBrowseRequest struct {
	Path string `json:"path" binding:"required"`
}

func (s *LocalServer) handleFilesBrowse(c *gin.Context) {
	var req filesBrowseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.runTool(c, "read_dir_list", map[string]interface{}{"path": req.Path})
}

type filesPreviewRequest struct {
	Path  string `json:"path" binding:"required"`
	Lines int    `json:"lines"`
}

func (s *LocalServer) handleFilesPreview(c *gin.Context) {
	var req filesPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lines := req.Lines
	if lines <= 0 {
		lines = 100
	}
	s.runTool(c, "read_file_tail", map[string]interface{}{
		"path":  req.Path,
		"lines": float64(lines),
	})
}

type filesDownloadRequest struct {
	Path      string  `json:"path" binding:"required"`
	UploadURL string  `json:"upload_url" binding:"required"`
	MaxSizeMB float64 `json:"max_size_mb"`
}

func (s *LocalServer) handleFilesDownload(c *gin.Context) {
	var req filesDownloadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	params := map[string]interface{}{
		"path":       req.Path,
		"upload_url": req.UploadURL,
	}
	if req.MaxSizeMB > 0 {
		params["max_size_mb"] = req.MaxSizeMB
	}
	s.runTool(c, "file_download", params)
}

func (s *LocalServer) runTool(c *gin.Context, toolName string, params map[string]interface{}) {
	if s.toolRegistry == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "tool registry not available"})
		return
	}
	tool, ok := s.toolRegistry.Get(toolName)
	if !ok {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("tool %q not registered", toolName)})
		return
	}
	result, err := tool.Execute(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
