package repository

import (
	"context"
	"time"

	"github.com/IanShaw027/sub2api-plugin-market/ent"
	"github.com/IanShaw027/sub2api-plugin-market/ent/syncjob"
	"github.com/google/uuid"
)

// SyncJobRepository 同步任务数据访问层
type SyncJobRepository struct {
	client *ent.Client
}

// NewSyncJobRepository 创建同步任务仓库
func NewSyncJobRepository(client *ent.Client) *SyncJobRepository {
	return &SyncJobRepository{client: client}
}

// ListSyncJobsParams 同步任务列表查询参数
type ListSyncJobsParams struct {
	Status      string
	PluginID    string
	TriggerType string
	Page        int
	PageSize    int
	From        *time.Time
	To          *time.Time
}

// Create 创建同步任务
func (r *SyncJobRepository) Create(ctx context.Context, pluginID uuid.UUID, triggerType, status, targetRef string) (*ent.SyncJob, error) {
	create := r.client.SyncJob.Create().
		SetPluginID(pluginID).
		SetTriggerType(syncjob.TriggerType(triggerType)).
		SetStatus(syncjob.Status(status))

	if targetRef != "" {
		create = create.SetTargetRef(targetRef)
	}

	return create.Save(ctx)
}

// GetByID 根据 ID 获取同步任务
func (r *SyncJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*ent.SyncJob, error) {
	return r.client.SyncJob.Get(ctx, id)
}

// UpdateStatus 更新同步任务状态
func (r *SyncJobRepository) UpdateStatus(ctx context.Context, job *ent.SyncJob, status string, errorMsg string, startedAt, finishedAt *time.Time) (*ent.SyncJob, error) {
	update := job.Update().SetStatus(syncjob.Status(status))

	if errorMsg != "" {
		update = update.SetErrorMessage(errorMsg)
	} else {
		update = update.ClearErrorMessage()
	}
	if startedAt != nil {
		update = update.SetStartedAt(*startedAt)
	}
	if finishedAt != nil {
		update = update.SetFinishedAt(*finishedAt)
	}

	return update.Save(ctx)
}

// List 分页查询同步任务列表
func (r *SyncJobRepository) List(ctx context.Context, params ListSyncJobsParams) ([]*ent.SyncJob, int, error) {
	query := r.client.SyncJob.Query()

	if params.Status != "" {
		query = query.Where(syncjob.StatusEQ(syncjob.Status(params.Status)))
	}

	if params.PluginID != "" {
		pluginUID, err := uuid.Parse(params.PluginID)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where(syncjob.PluginIDEQ(pluginUID))
	}

	if params.TriggerType != "" {
		query = query.Where(syncjob.TriggerTypeEQ(syncjob.TriggerType(params.TriggerType)))
	}

	if params.From != nil {
		query = query.Where(syncjob.CreatedAtGTE(*params.From))
	}

	if params.To != nil {
		query = query.Where(syncjob.CreatedAtLTE(*params.To))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	jobs, err := query.
		Order(ent.Desc(syncjob.FieldCreatedAt)).
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// CountRecentByPluginAndRef 统计指定时间窗口内、插件+引用+触发类型的任务数量
func (r *SyncJobRepository) CountRecentByPluginAndRef(ctx context.Context, pluginID uuid.UUID, triggerType, targetRef string, since time.Time) (int, error) {
	query := r.client.SyncJob.Query().
		Where(
			syncjob.PluginIDEQ(pluginID),
			syncjob.TriggerTypeEQ(syncjob.TriggerType(triggerType)),
			syncjob.TargetRefEQ(targetRef),
			syncjob.StatusIn(syncjob.StatusPending, syncjob.StatusRunning, syncjob.StatusSucceeded),
			syncjob.CreatedAtGTE(since),
		)
	return query.Count(ctx)
}
