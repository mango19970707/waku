package constant

import "fmt"

// 文件类型常量
const (
	// FileTypeOutline 大纲文件类型
	FileTypeOutline = "outline"
	// FileTypePlotline 剧情线文件类型
	FileTypePlotline = "plotline"
	// FileTypeVolume 分卷文件类型
	FileTypeVolume = "volume"
	// FileTypeChapter 章节文件类型
	FileTypeChapter = "chapter"
)

// 大纲处理阶段常量
const (
	// StageGenerate 生成阶段
	StageGenerate = "生成"
	// StageExpand 扩写阶段
	StageExpand = "扩写"
	// StagePolish 润色阶段
	StagePolish = "润色"
	// StageRevise 修改阶段
	StageRevise = "修改"
	// StageExtract 提取阶段（用于剧情线提取）
	StageExtract = "提取"
)

// 分卷处理阶段常量
const (
	// StagePartition 分卷阶段
	StagePartition = "分卷"
)

// 工作流配置常量
const (
	// MaxRetries 最大重试次数
	MaxRetries = 3
)

// 文件路径常量
const (
	// DataDir 数据根目录
	DataDir = "../../data"
	// FileExtension 文件扩展名
	FileExtension = ".md"
)

// 小说状态常量
const (
	// StatusDraft 草稿状态
	StatusDraft = "draft"
	// StatusWriting 写作中状态
	StatusWriting = "writing"
	// StatusCompleted 已完成状态
	StatusCompleted = "completed"
)

// 工作流节点状态常量
const (
	// NodeInit 初始化节点
	NodeInit = "init"
	// NodeSuccess 成功状态后缀
	NodeSuccess = "_success"
	// NodeFailed 失败状态后缀
	NodeFailed = "_failed"
	// NodeWaitingApproval 等待审批状态后缀
	NodeWaitingApproval = "_waiting_approval"
	// NodeRejected 拒绝状态后缀
	NodeRejected = "_rejected"
)

// FormatChapterStage 格式化章节阶段名称
//
// 参数：
//   - volumeNumber: 卷号
//
// 返回值：
//   - string: 格式化后的阶段名称，如"第1卷-章节"
func FormatChapterStage(volumeNumber int) string {
	return fmt.Sprintf("第%d卷-章节", volumeNumber)
}
