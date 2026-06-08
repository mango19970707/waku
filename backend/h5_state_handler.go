package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"waku/internal/model"
)

type h5StateHandler struct {
	db *gorm.DB
}

type h5StepConfig struct {
	Key       StepKey `json:"key"`
	Name      string  `json:"name"`
	DependsOn StepKey `json:"depends_on,omitempty"`
}

type draftPayload struct {
	Content map[string]any `json:"content"`
}

var h5Steps = []h5StepConfig{
	{Key: StepOutline, Name: "小说大纲"},
	{Key: StepPlotlines, Name: "剧情线", DependsOn: StepOutline},
	{Key: StepCharacters, Name: "关键人物设定", DependsOn: StepPlotlines},
	{Key: StepVolumes, Name: "分卷", DependsOn: StepCharacters},
	{Key: StepChapters, Name: "章节划分", DependsOn: StepVolumes},
	{Key: StepContent, Name: "正文撰写", DependsOn: StepChapters},
}

func newH5StateHandler(db *gorm.DB) *h5StateHandler {
	return &h5StateHandler{db: db}
}

func (h *h5StateHandler) me(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"id": 1, "nickname": "创作者昵称", "avatar_url": "", "bio": "这里将展示用户简介"})
}

func (h *h5StateHandler) meStats(c *gin.Context) {
	if h.db == nil {
		c.JSON(http.StatusOK, gin.H{"novel_count": 0, "word_count": 0})
		return
	}
	var novelCount int64
	var wordCount int64
	// 统计接口仅聚合当前本地库数据，避免个人中心因为无登录体系而依赖外部用户表。
	h.db.Model(&model.Novel{}).Count(&novelCount)
	h.db.Model(&model.Chapter{}).Select("COALESCE(SUM(word_count), 0)").Scan(&wordCount)
	c.JSON(http.StatusOK, gin.H{"novel_count": novelCount, "word_count": wordCount})
}

func (h *h5StateHandler) listNovels(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	var novels []model.Novel
	if err := h.db.Order("updated_at DESC, id DESC").Find(&novels).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "获取小说列表失败")
		return
	}
	items := make([]map[string]any, 0, len(novels))
	for _, novel := range novels {
		archivedCount := h.archivedCount(novel.ID)
		items = append(items, map[string]any{
			"id":             novel.ID,
			"title":          novel.Title,
			"genre":          novel.Genre,
			"updated_at":     novel.UpdatedAt,
			"current_step":   h.currentStepKey(archivedCount),
			"archived_count": archivedCount,
		})
	}
	c.JSON(http.StatusOK, gin.H{"novels": items})
}

func (h *h5StateHandler) createNovel(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	var req struct {
		Title           string `json:"title"`
		Author          string `json:"author"`
		Genre           string `json:"genre"`
		OriginalOutline string `json:"original_outline"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(c, http.StatusBadRequest, "小说标题不能为空")
		return
	}
	if strings.TrimSpace(req.Author) == "" {
		req.Author = "默认作者"
	}
	novel := model.Novel{Title: req.Title, Author: req.Author, Genre: req.Genre, Synopsis: req.OriginalOutline, Status: "draft"}
	// 创建小说只初始化基础记录；六个步骤的状态由现有内容实时推导，减少额外状态表依赖。
	if err := h.db.Create(&novel).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "创建小说失败")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"novel": novel, "workflow": h.workflowPayload(novel.ID)})
}

func (h *h5StateHandler) deleteNovel(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	novelID, ok := parseUintParam(c, "novel_id", "小说ID格式不正确")
	if !ok {
		return
	}
	// 删除小说使用 GORM 软删除，保留关联内容便于后续恢复和审计。
	if err := h.db.Delete(&model.Novel{}, novelID).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "删除小说失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "小说已删除"})
}

func (h *h5StateHandler) getWorkflow(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	novelID, ok := parseUintParam(c, "novel_id", "小说ID格式不正确")
	if !ok {
		return
	}
	c.JSON(http.StatusOK, h.workflowPayload(novelID))
}

func (h *h5StateHandler) getStep(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	novelID, ok := parseUintParam(c, "novel_id", "小说ID格式不正确")
	if !ok {
		return
	}
	step, ok := parseStepKey(c.Param("step"))
	if !ok {
		writeError(c, http.StatusBadRequest, "不支持的创作步骤")
		return
	}
	payload, err := h.stepPayload(novelID, step)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, payload)
}

func (h *h5StateHandler) saveStepDraft(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	novelID, ok := parseUintParam(c, "novel_id", "小说ID格式不正确")
	if !ok {
		return
	}
	step, ok := parseStepKey(c.Param("step"))
	if !ok {
		writeError(c, http.StatusBadRequest, "不支持的创作步骤")
		return
	}
	var req draftPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	if err := h.applyStepDraft(novelID, step, req.Content); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "草稿已保存", "step": step})
}

func (h *h5StateHandler) editArchivedStep(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "已进入归档编辑模式，下游步骤可能需要重新检查", "editing_archived": true})
}

func (h *h5StateHandler) rearchiveStep(c *gin.Context) {
	step, ok := parseStepKey(c.Param("step"))
	if !ok {
		writeError(c, http.StatusBadRequest, "不支持的创作步骤")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "重新归档完成，请检查所有下游步骤", "step": step})
}

func (h *h5StateHandler) listMessages(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"messages": []any{}})
}

func (h *h5StateHandler) streamMessages(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.String(http.StatusOK, "event: done\ndata: {}\n\n")
}

func (h *h5StateHandler) stopMessage(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "已停止生成"})
}

func (h *h5StateHandler) getVolumeChapters(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	novelID, novelOK := parseUintParam(c, "novel_id", "小说ID格式不正确")
	volumeID, volumeOK := parseUintParam(c, "volume_id", "分卷ID格式不正确")
	if !novelOK || !volumeOK {
		return
	}
	var chapters []model.Chapter
	if err := h.db.Where("novel_id = ? AND volume_id = ?", novelID, volumeID).Order("chapter_number ASC, id ASC").Find(&chapters).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "获取分卷章节失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"chapters": chapters})
}

func (h *h5StateHandler) getChapterContent(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	chapterID, ok := parseUintParam(c, "chapter_id", "章节ID格式不正确")
	if !ok {
		return
	}
	var chapter model.Chapter
	if err := h.db.First(&chapter, chapterID).Error; err != nil {
		writeError(c, http.StatusNotFound, "章节不存在")
		return
	}
	c.JSON(http.StatusOK, gin.H{"chapter": chapter, "review_suggestions": []any{}})
}

func (h *h5StateHandler) saveContentDraft(c *gin.Context) {
	if !h.ensureDB(c) {
		return
	}
	chapterID, ok := parseUintParam(c, "chapter_id", "章节ID格式不正确")
	if !ok {
		return
	}
	var req struct {
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "请求参数格式不正确")
		return
	}
	if strings.TrimSpace(req.Content) == "" {
		writeError(c, http.StatusBadRequest, "正文不能为空")
		return
	}
	if req.Status == "" {
		req.Status = "writing"
	}
	// 正文草稿只改正文相关字段，不影响章节规划字段。
	updates := map[string]any{"content": req.Content, "word_count": len([]rune(req.Content)), "status": req.Status, "updated_at": time.Now()}
	if err := h.db.Model(&model.Chapter{}).Where("id = ?", chapterID).Updates(updates).Error; err != nil {
		writeError(c, http.StatusInternalServerError, "保存正文草稿失败")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "正文草稿已保存"})
}

func (h *h5StateHandler) archiveContent(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "章节正文已归档"})
}

func (h *h5StateHandler) transcribeSpeech(c *gin.Context) {
	var req struct {
		Text string `json:"text"`
	}
	if c.Request.ContentLength > 0 {
		_ = c.ShouldBindJSON(&req)
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		text = "帮我增加一个亦敌亦友角色"
	}
	c.JSON(http.StatusOK, gin.H{"text": text})
}

func (h *h5StateHandler) workflowPayload(novelID uint) gin.H {
	archivedCount := h.archivedCount(novelID)
	steps := make([]map[string]any, 0, len(h5Steps))
	for index, item := range h5Steps {
		status := "locked"
		if index < archivedCount {
			status = "archived"
		} else if index == archivedCount {
			status = "active"
		}
		steps = append(steps, map[string]any{"key": item.Key, "name": item.Name, "depends_on": item.DependsOn, "status": status})
	}
	return gin.H{"novel_id": novelID, "archived_count": archivedCount, "current_step": h.currentStepKey(archivedCount), "steps": steps}
}

func (h *h5StateHandler) stepPayload(novelID uint, step StepKey) (gin.H, error) {
	switch step {
	case StepOutline:
		var novel model.Novel
		if err := h.db.First(&novel, novelID).Error; err != nil {
			return nil, fmt.Errorf("小说不存在")
		}
		return gin.H{"step": step, "draft": novel, "archived": h.hasOutline(novel), "upstream_summary": nil, "review_suggestions": []any{}}, nil
	case StepPlotlines:
		var plotlines []model.Plotline
		h.db.Where("novel_id = ?", novelID).Order("type ASC, id ASC").Find(&plotlines)
		return gin.H{"step": step, "draft": gin.H{"plotlines": plotlines}, "archived": len(plotlines) > 0, "upstream_summary": h.outlineSummary(novelID), "review_suggestions": []any{}}, nil
	case StepCharacters:
		var characters []model.Character
		h.db.Where("novel_id = ?", novelID).Order("id ASC").Find(&characters)
		return gin.H{"step": step, "draft": gin.H{"characters": characters}, "archived": len(characters) > 0, "upstream_summary": h.outlineSummary(novelID), "review_suggestions": []any{}}, nil
	case StepVolumes:
		var volumes []model.Volume
		h.db.Where("novel_id = ?", novelID).Order("volume_number ASC, id ASC").Find(&volumes)
		return gin.H{"step": step, "draft": gin.H{"volumes": volumes}, "archived": len(volumes) > 0, "upstream_summary": h.outlineSummary(novelID), "review_suggestions": []any{}}, nil
	case StepChapters, StepContent:
		return h.chapterStepPayload(novelID, step), nil
	default:
		return nil, fmt.Errorf("不支持的创作步骤")
	}
}

func (h *h5StateHandler) chapterStepPayload(novelID uint, step StepKey) gin.H {
	var volumes []model.Volume
	h.db.Where("novel_id = ?", novelID).Order("volume_number ASC, id ASC").Find(&volumes)
	var chapters []model.Chapter
	h.db.Where("novel_id = ?", novelID).Order("volume_id ASC, chapter_number ASC, id ASC").Find(&chapters)
	return gin.H{"step": step, "draft": gin.H{"volumes": volumes, "chapters": chapters}, "archived": len(chapters) > 0, "upstream_summary": h.outlineSummary(novelID), "review_suggestions": []any{}}
}

func (h *h5StateHandler) applyStepDraft(novelID uint, step StepKey, content map[string]any) error {
	if len(content) == 0 {
		return fmt.Errorf("草稿内容不能为空")
	}
	switch step {
	case StepOutline:
		updates := map[string]any{"updated_at": time.Now()}
		if value, ok := stringValue(content, "title"); ok {
			updates["title"] = value
		}
		if value, ok := stringValue(content, "synopsis"); ok {
			updates["synopsis"] = value
		}
		outline, _ := json.Marshal(content)
		updates["outline"] = string(outline)
		return h.db.Model(&model.Novel{}).Where("id = ?", novelID).Updates(updates).Error
	default:
		return fmt.Errorf("当前步骤草稿保存需要使用对应工作流生成接口")
	}
}

func (h *h5StateHandler) archivedCount(novelID uint) int {
	count := 0
	if h.db == nil {
		return count
	}
	var novel model.Novel
	if err := h.db.First(&novel, novelID).Error; err == nil && h.hasOutline(novel) {
		count = 1
	}
	var total int64
	h.db.Model(&model.Plotline{}).Where("novel_id = ?", novelID).Count(&total)
	if total > 0 && count == 1 {
		count = 2
	}
	h.db.Model(&model.Character{}).Where("novel_id = ?", novelID).Count(&total)
	if total > 0 && count == 2 {
		count = 3
	}
	h.db.Model(&model.Volume{}).Where("novel_id = ?", novelID).Count(&total)
	if total > 0 && count == 3 {
		count = 4
	}
	h.db.Model(&model.Chapter{}).Where("novel_id = ?", novelID).Count(&total)
	if total > 0 && count == 4 {
		count = 5
	}
	h.db.Model(&model.Chapter{}).Where("novel_id = ? AND content <> ''", novelID).Count(&total)
	if total > 0 && count == 5 {
		count = 6
	}
	return count
}

func (h *h5StateHandler) currentStepKey(archivedCount int) StepKey {
	if archivedCount >= len(h5Steps) {
		return StepContent
	}
	return h5Steps[archivedCount].Key
}

func (h *h5StateHandler) hasOutline(novel model.Novel) bool {
	return strings.TrimSpace(novel.Title) != "" && strings.TrimSpace(novel.Synopsis) != "" && strings.TrimSpace(novel.Outline) != ""
}

func (h *h5StateHandler) outlineSummary(novelID uint) gin.H {
	var novel model.Novel
	if h.db == nil || h.db.First(&novel, novelID).Error != nil {
		return gin.H{}
	}
	return gin.H{"title": novel.Title, "genre": novel.Genre, "synopsis": novel.Synopsis, "outline": novel.Outline}
}

func (h *h5StateHandler) ensureDB(c *gin.Context) bool {
	if h.db == nil {
		writeError(c, http.StatusServiceUnavailable, "数据库未配置，无法读取创作数据")
		return false
	}
	return true
}

func stringValue(content map[string]any, key string) (string, bool) {
	value, ok := content[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	return strings.TrimSpace(text), ok
}
