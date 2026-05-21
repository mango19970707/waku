package backend

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Config struct {
	WorkflowRunner WorkflowRunner
}

type errorResponse struct {
	Message string `json:"message"`
}

func NewRouter(cfg Config) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	runner := cfg.WorkflowRunner
	if runner == nil {
		runner = &notConfiguredRunner{}
	}

	api := router.Group("/api/h5")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	api.POST("/novels/:novel_id/steps/:step/messages", runStepHandler(runner))
	api.POST("/novels/:novel_id/steps/:step/generate", runStepHandler(runner))
	api.POST("/novels/:novel_id/steps/:step/archive", runStepHandler(runner))
	api.POST("/novels/:novel_id/volumes/:volume_id/chapters/messages", runNestedStepHandler(runner, StepChapters))
	api.POST("/novels/:novel_id/volumes/:volume_id/chapters/generate", runNestedStepHandler(runner, StepChapters))
	api.POST("/chapters/:chapter_id/content/messages", runChapterContentHandler(runner))
	api.POST("/chapters/:chapter_id/content/generate", runChapterContentHandler(runner))
	api.POST("/novels/:novel_id/volumes/:volume_id/content/batch-generate", runNestedStepHandler(runner, StepContent))

	return router
}

func runStepHandler(runner WorkflowRunner) gin.HandlerFunc {
	return func(c *gin.Context) {
		step, ok := parseStepKey(c.Param("step"))
		if !ok {
			writeError(c, http.StatusBadRequest, "不支持的创作步骤")
			return
		}
		req, ok := bindWorkflowRequest(c)
		if !ok {
			return
		}
		novelID, ok := parseUintParam(c, "novel_id", "小说ID格式不正确")
		if !ok {
			return
		}
		req.Step = step
		req.NovelID = novelID
		executeWorkflow(c, runner, req)
	}
}

func runNestedStepHandler(runner WorkflowRunner, step StepKey) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, ok := bindWorkflowRequest(c)
		if !ok {
			return
		}
		novelID, ok := parseUintParam(c, "novel_id", "小说ID格式不正确")
		if !ok {
			return
		}
		volumeID, ok := parseUintParam(c, "volume_id", "分卷ID格式不正确")
		if !ok {
			return
		}
		req.Step = step
		req.NovelID = novelID
		req.VolumeID = volumeID
		executeWorkflow(c, runner, req)
	}
}

func runChapterContentHandler(runner WorkflowRunner) gin.HandlerFunc {
	return func(c *gin.Context) {
		req, ok := bindWorkflowRequest(c)
		if !ok {
			return
		}
		chapterID, ok := parseUintParam(c, "chapter_id", "章节ID格式不正确")
		if !ok {
			return
		}
		req.Step = StepContent
		req.ChapterID = chapterID
		executeWorkflow(c, runner, req)
	}
}

func executeWorkflow(c *gin.Context, runner WorkflowRunner, req WorkflowRunRequest) {
	result, err := runner.Run(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusBadGateway, "工作流执行失败，请稍后重试")
		return
	}
	c.JSON(http.StatusOK, result)
}

func bindWorkflowRequest(c *gin.Context) (WorkflowRunRequest, bool) {
	var req WorkflowRunRequest
	if c.Request.Body == nil || c.Request.ContentLength == 0 {
		return req, true
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求参数格式不正确")
		return req, false
	}
	req.UserSuggestion = strings.TrimSpace(req.UserSuggestion)
	return req, true
}

func parseStepKey(raw string) (StepKey, bool) {
	switch StepKey(raw) {
	case StepOutline, StepPlotlines, StepCharacters, StepVolumes, StepChapters, StepContent:
		return StepKey(raw), true
	default:
		return "", false
	}
}

func parseUintParam(c *gin.Context, name, message string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || value == 0 {
		writeError(c, http.StatusBadRequest, message)
		return 0, false
	}
	return uint(value), true
}

func writeError(c *gin.Context, status int, message string) {
	c.JSON(status, errorResponse{Message: message})
}

type notConfiguredRunner struct{}

func (r *notConfiguredRunner) Run(_ context.Context, _ WorkflowRunRequest) (*WorkflowRunResult, error) {
	return nil, errors.New("workflow runner 未配置")
}
