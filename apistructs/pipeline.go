package apistructs

import "time"

type PipelineStatus string

const (
	PipelineEmptyStatus PipelineStatus = "" // 判断状态是否为空

	// 构建相关的状态
	PipelineStatusInitializing  PipelineStatus = "Initializing"  // 初始化中：存在时间一般来说极短，表示 build 刚创建并正在分析中
	PipelineStatusDisabled      PipelineStatus = "Disabled"      // 禁用状态：表示该节点被禁用
	PipelineStatusAnalyzeFailed PipelineStatus = "AnalyzeFailed" // 分析失败：分析结束但是结果失败
	PipelineStatusAnalyzed      PipelineStatus = "Analyzed"      // 分析完毕：build 创建完即开始分析，分析成功则为该状态

	// 流程推进相关的状态
	PipelineStatusBorn    PipelineStatus = "Born"    // 流程推进过程中的初始状态
	PipelineStatusPaused  PipelineStatus = "Paused"  // 暂停状态：表示流程需要暂停，和 Born 同级，不会被 Mark
	PipelineStatusMark    PipelineStatus = "Mark"    // 标记状态：表示流程开始处理
	PipelineStatusCreated PipelineStatus = "Created" // 创建成功：scheduler create + start；可能要区分 Created 和 Started 两个状态
	PipelineStatusQueue   PipelineStatus = "Queue"   // 排队中：介于 启动成功 和 运行中
	PipelineStatusRunning PipelineStatus = "Running" // 运行中
	PipelineStatusSuccess PipelineStatus = "Success" // 成功

	// 流程推进 "正常" 失败：一般是用户侧导致的失败
	PipelineStatusFailed         PipelineStatus = "Failed"         // 业务逻辑执行失败，"正常" 失败
	PipelineStatusTimeout        PipelineStatus = "Timeout"        // 超时
	PipelineStatusStopByUser     PipelineStatus = "StopByUser"     // 用户主动取消
	PipelineStatusNoNeedBySystem PipelineStatus = "NoNeedBySystem" // 无需执行：系统判定无需执行

	// 流程推进 "异常" 失败：一般是平台侧导致的失败
	PipelineStatusCreateError    PipelineStatus = "CreateError"    // 创建节点失败
	PipelineStatusStartError     PipelineStatus = "StartError"     // 开始节点失败
	PipelineStatusError          PipelineStatus = "Error"          // 异常
	PipelineStatusDBError        PipelineStatus = "DBError"        // 平台流程推进时操作数据库异常
	PipelineStatusUnknown        PipelineStatus = "Unknown"        // 未知状态：获取到了无法识别的状态，流程无法推进
	PipelineStatusLostConn       PipelineStatus = "LostConn"       // 在重试指定次数后仍然无法连接
	PipelineStatusCancelByRemote PipelineStatus = "CancelByRemote" // 远端取消

	// 人工审核相关
	PipelineStatusWaitApproval    PipelineStatus = "WaitApprove" // 等待人工审核
	PipelineStatusApprovalSuccess PipelineStatus = "Accept"      // 人工审核通过
	PipelineStatusApprovalFail    PipelineStatus = "Reject"      // 人工审核拒绝
)

func (status PipelineStatus) IsSuccessStatus() bool {
	return status == PipelineStatusSuccess
}

func (status PipelineStatus) IsFailedStatus() bool {
	return status.IsNormalFailedStatus() || status.IsAbnormalFailedStatus()
}

// IsNormalFailedStatus 表示正常失败，一般由用户侧引起
func (status PipelineStatus) IsNormalFailedStatus() bool {
	switch status {
	// "正常" 失败
	case PipelineStatusAnalyzeFailed, PipelineStatusFailed, PipelineStatusTimeout,
		PipelineStatusStopByUser, PipelineStatusNoNeedBySystem:
		return true
	default:
		return false
	}
}

// IsAbnormalFailedStatus 表示异常失败，一般由平台侧引起
func (status PipelineStatus) IsAbnormalFailedStatus() bool {
	switch status {
	// "异常" 失败
	case PipelineStatusCreateError, PipelineStatusStartError, PipelineStatusDBError,
		PipelineStatusError, PipelineStatusUnknown, PipelineStatusLostConn, PipelineStatusCancelByRemote:
		return true
	default:
		return false
	}
}

type ErrorResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Ctx  interface{} `json:"ctx"`
}

type Header struct {
	Success bool          `json:"success" `
	Error   ErrorResponse `json:"err"`
}

type GetPipelineResponse struct {
	Header
	Data PipelineDetailDTO `json:"data"`
}

type PipelineDTO struct {
	// 应用相关信息
	ID              uint64  `json:"id,omitempty"`
	CronID          *uint64 `json:"cronID,omitempty"`
	OrgID           uint64  `json:"orgID,omitempty"`
	OrgName         string  `json:"orgName,omitempty"`
	ProjectID       uint64  `json:"projectID,omitempty"`
	ProjectName     string  `json:"projectName,omitempty"`
	ApplicationID   uint64  `json:"applicationID,omitempty"`
	ApplicationName string  `json:"applicationName,omitempty"`

	// 运行时相关信息
	Namespace   string         `json:"namespace"`
	Type        string         `json:"type,omitempty"`
	TriggerMode string         `json:"triggerMode,omitempty"`
	ClusterName string         `json:"clusterName,omitempty"`
	Status      PipelineStatus `json:"status,omitempty"`
	Progress    float64        `json:"progress"` // pipeline 执行进度, eg: 0.8 即 80%

	// 时间
	CostTimeSec int64      `json:"costTimeSec,omitempty"`                // pipeline 总耗时/秒
	TimeBegin   *time.Time `json:"timeBegin,omitempty"`                  // 执行开始时间
	TimeEnd     *time.Time `json:"timeEnd,omitempty"`                    // 执行结束时间
	TimeCreated *time.Time `json:"timeCreated,omitempty" xorm:"created"` // 记录创建时间
	TimeUpdated *time.Time `json:"timeUpdated,omitempty" xorm:"updated"` // 记录更新时间
}

// PipelineDetailDTO contains pipeline, stages, tasks and others
type PipelineDetailDTO struct {
	PipelineDTO
	PipelineStages []PipelineStageDetailDTO `json:"pipelineStages"`
}

type PipelineStageDetailDTO struct {
	PipelineStageDTO
	PipelineTasks []PipelineTaskDTO `json:"pipelineTasks"`
}

type PipelineStageDTO struct {
	ID         uint64 `json:"id"`
	PipelineID uint64 `json:"pipelineID"`

	Name   string         `json:"name"`
	Status PipelineStatus `json:"status"`

	CostTimeSec int64     `json:"costTimeSec"`
	TimeBegin   time.Time `json:"timeBegin"`
	TimeEnd     time.Time `json:"timeEnd"`
	TimeCreated time.Time `json:"timeCreated"`
	TimeUpdated time.Time `json:"timeUpdated"`
}

type PipelineTaskDTO struct {
	ID         uint64 `json:"id"`
	PipelineID uint64 `json:"pipelineID"`
	StageID    uint64 `json:"stageID"`

	Name   string             `json:"name"`
	OpType string             `json:"opType"`         // get, put, task
	Type   string             `json:"type,omitempty"` // git, buildpack, release, dice ... 当 OpType 为自定义任务时为空
	Status PipelineStatus     `json:"status"`
	Labels map[string]string  `json:"labels"`
	Result PipelineTaskResult `json:"result"`

	CostTimeSec  int64     `json:"costTimeSec"`  // -1 表示暂无耗时信息, 0 表示确实是0s结束
	QueueTimeSec int64     `json:"queueTimeSec"` // 等待调度的耗时, -1 暂无耗时信息, 0 表示确实是0s结束 TODO 赋值
	TimeBegin    time.Time `json:"timeBegin"`    // 执行开始时间
	TimeEnd      time.Time `json:"timeEnd"`      // 执行结束时间
	TimeCreated  time.Time `json:"timeCreated"`  // 记录创建时间
	TimeUpdated  time.Time `json:"timeUpdated"`  // 记录更新时间
}

type PipelineTaskResult struct {
	Errors []ErrorResponse `json:"errors,omitempty"`
}
