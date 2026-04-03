package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/domain"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/dto"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/infrastructure"
	"github.com/zy-eagle/envnexus/services/platform-api/internal/repository"
)

type Service struct {
	taskRepo      repository.CommandTaskRepository
	execRepo      repository.CommandExecutionRepository
	policyService *ApprovalPolicyService
	auditRepo     repository.AuditRepository
	rbacService   PermissionChecker
	gatewayClient *infrastructure.GatewayClient
}

type PermissionChecker interface {
	HasPermission(ctx context.Context, userID, permission string) (bool, error)
}

func NewService(
	taskRepo repository.CommandTaskRepository,
	execRepo repository.CommandExecutionRepository,
	policyService *ApprovalPolicyService,
	auditRepo repository.AuditRepository,
	rbacService PermissionChecker,
	gatewayClient *infrastructure.GatewayClient,
) *Service {
	return &Service{
		taskRepo:      taskRepo,
		execRepo:      execRepo,
		policyService: policyService,
		auditRepo:     auditRepo,
		rbacService:   rbacService,
		gatewayClient: gatewayClient,
	}
}

func (s *Service) CreateTask(ctx context.Context, tenantID, userID string, req dto.CreateCommandTaskRequest) (*dto.CommandTaskResponse, error) {
	systemRisk := EvaluateRisk(req.CommandType, req.CommandPayload)
	effectiveRisk := EffectiveRisk(req.RiskLevel, systemRisk)

	deviceIDsJSON, _ := json.Marshal(req.DeviceIDs)

	policy, _ := s.policyService.FindPolicy(ctx, tenantID, effectiveRisk)

	expiresMinutes := 30
	if policy != nil && policy.ExpiresMinutes > 0 {
		expiresMinutes = policy.ExpiresMinutes
	}

	task := &domain.CommandTask{
		ID:              ulid.Make().String(),
		TenantID:        tenantID,
		CreatedByUserID: userID,
		Title:           req.Title,
		CommandType:     req.CommandType,
		CommandPayload:  req.CommandPayload,
		DeviceIDsJSON:   string(deviceIDsJSON),
		RiskLevel:       req.RiskLevel,
		EffectiveRisk:   effectiveRisk,
		Emergency:       req.Emergency,
		BypassReason:    req.BypassReason,
		TargetEnv:       req.TargetEnv,
		ChangeTicket:    req.ChangeTicket,
		BusinessApp:     req.BusinessApp,
		Note:            req.Note,
		ExpiresAt:       time.Now().Add(time.Duration(expiresMinutes) * time.Minute),
	}

	if policy != nil {
		task.PolicySnapshotID = &policy.ID
	}

	for _, deviceID := range req.DeviceIDs {
		exec := &domain.CommandExecution{
			ID:       ulid.Make().String(),
			TaskID:   task.ID,
			DeviceID: deviceID,
			Status:   domain.ExecutionPending,
		}
		if err := s.execRepo.Create(ctx, exec); err != nil {
			return nil, fmt.Errorf("create execution record: %w", err)
		}
	}

	bypass := s.shouldBypassApproval(ctx, tenantID, userID, req, effectiveRisk, policy)

	if bypass {
		task.BypassApproval = true
		task.Status = domain.CommandTaskApproved
		now := time.Now()
		task.ApprovedAt = &now
		if err := s.taskRepo.Create(ctx, task); err != nil {
			return nil, err
		}
		s.recordAudit(ctx, tenantID, task.ID, userID, "command.emergency_bypass", map[string]interface{}{
			"bypass_reason": req.BypassReason,
			"risk_level":    effectiveRisk,
			"device_count":  len(req.DeviceIDs),
		})
		go s.dispatchToDevices(context.Background(), task)
		return s.taskToResponse(ctx, task)
	}

	if policy != nil && policy.AutoApprove {
		task.Status = domain.CommandTaskApproved
		now := time.Now()
		task.ApprovedAt = &now
		if err := s.taskRepo.Create(ctx, task); err != nil {
			return nil, err
		}
		go s.dispatchToDevices(context.Background(), task)
		return s.taskToResponse(ctx, task)
	}

	approverUserID, approverRoleID := s.resolveApprover(ctx, policy)
	task.ApproverUserID = approverUserID
	task.ApproverRoleID = approverRoleID
	task.Status = domain.CommandTaskPendingApproval

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, tenantID, task.ID, userID, "command.task_created", map[string]interface{}{
		"risk_level":   effectiveRisk,
		"device_count": len(req.DeviceIDs),
		"command_type": req.CommandType,
	})

	return s.taskToResponse(ctx, task)
}

func (s *Service) UpdateTask(ctx context.Context, tenantID, userID, taskID string, req dto.UpdateCommandTaskRequest) (*dto.CommandTaskResponse, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil || task.TenantID != tenantID {
		return nil, domain.ErrCommandTaskNotFound
	}
	if task.Status != domain.CommandTaskPendingApproval {
		return nil, domain.ErrCommandTaskInvalidState
	}
	if time.Now().After(task.ExpiresAt) {
		task.Status = domain.CommandTaskExpired
		_ = s.taskRepo.Update(ctx, task)
		return nil, domain.ErrCommandTaskExpired
	}
	if task.CreatedByUserID != userID {
		return nil, domain.ErrInsufficientPermission
	}

	systemRisk := EvaluateRisk(req.CommandType, req.CommandPayload)
	effectiveRisk := EffectiveRisk(req.RiskLevel, systemRisk)

	policy, _ := s.policyService.FindPolicy(ctx, tenantID, effectiveRisk)
	expiresMinutes := 30
	if policy != nil && policy.ExpiresMinutes > 0 {
		expiresMinutes = policy.ExpiresMinutes
	}

	// Update task fields
	task.Title = req.Title
	task.CommandType = req.CommandType
	task.CommandPayload = req.CommandPayload
	task.SetDeviceIDs(req.DeviceIDs)
	task.RiskLevel = req.RiskLevel
	task.EffectiveRisk = effectiveRisk
	task.Emergency = req.Emergency
	task.BypassReason = req.BypassReason
	task.TargetEnv = req.TargetEnv
	task.ChangeTicket = req.ChangeTicket
	task.BusinessApp = req.BusinessApp
	task.Note = req.Note
	task.ExpiresAt = time.Now().Add(time.Duration(expiresMinutes) * time.Minute)

	// Reset approval-related fields
	task.BypassApproval = false
	task.PolicySnapshotID = nil
	task.ApproverUserID = nil
	task.ApproverRoleID = nil
	task.ApprovedByID = nil
	task.ApprovalNote = ""
	task.ApprovedAt = nil

	if policy != nil {
		task.PolicySnapshotID = &policy.ID
	}

	// Rebuild executions for new device list
	if err := s.execRepo.DeleteByTaskID(ctx, task.ID); err != nil {
		return nil, fmt.Errorf("delete old executions: %w", err)
	}
	for _, deviceID := range req.DeviceIDs {
		exec := &domain.CommandExecution{
			ID:       ulid.Make().String(),
			TaskID:   task.ID,
			DeviceID: deviceID,
			Status:   domain.ExecutionPending,
		}
		if err := s.execRepo.Create(ctx, exec); err != nil {
			return nil, fmt.Errorf("create execution record: %w", err)
		}
	}

	bypass := s.shouldBypassApproval(ctx, tenantID, userID, dto.CreateCommandTaskRequest{
		Title:          req.Title,
		CommandType:    req.CommandType,
		CommandPayload: req.CommandPayload,
		DeviceIDs:      req.DeviceIDs,
		RiskLevel:      req.RiskLevel,
		Emergency:      req.Emergency,
		BypassReason:   req.BypassReason,
		TargetEnv:      req.TargetEnv,
		ChangeTicket:   req.ChangeTicket,
		BusinessApp:    req.BusinessApp,
		Note:           req.Note,
	}, effectiveRisk, policy)

	if bypass {
		task.BypassApproval = true
		task.Status = domain.CommandTaskApproved
		now := time.Now()
		task.ApprovedAt = &now
	} else if policy != nil && policy.AutoApprove {
		task.Status = domain.CommandTaskApproved
		now := time.Now()
		task.ApprovedAt = &now
	} else {
		approverUserID, approverRoleID := s.resolveApprover(ctx, policy)
		task.ApproverUserID = approverUserID
		task.ApproverRoleID = approverRoleID
		task.Status = domain.CommandTaskPendingApproval
	}

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, err
	}

	s.recordAudit(ctx, tenantID, task.ID, userID, "command.task_updated", map[string]interface{}{
		"risk_level":   effectiveRisk,
		"device_count": len(req.DeviceIDs),
		"command_type": req.CommandType,
	})

	if task.Status == domain.CommandTaskApproved {
		go s.dispatchToDevices(context.Background(), task)
	}
	return s.taskToResponse(ctx, task)
}

func (s *Service) DeleteTask(ctx context.Context, tenantID, userID, taskID string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil || task.TenantID != tenantID {
		return domain.ErrCommandTaskNotFound
	}
	if task.CreatedByUserID != userID {
		return domain.ErrInsufficientPermission
	}
	// Only allow deleting tasks that haven't been executed yet.
	if task.Status != domain.CommandTaskPendingApproval && task.Status != domain.CommandTaskDenied && task.Status != domain.CommandTaskCancelled {
		return domain.ErrCommandTaskInvalidState
	}

	if err := s.execRepo.DeleteByTaskID(ctx, task.ID); err != nil {
		return fmt.Errorf("delete executions: %w", err)
	}
	if err := s.taskRepo.Delete(ctx, task.ID); err != nil {
		return err
	}
	s.recordAudit(ctx, tenantID, task.ID, userID, "command.task_deleted", map[string]interface{}{
		"status": string(task.Status),
	})
	return nil
}

func (s *Service) shouldBypassApproval(ctx context.Context, tenantID, userID string, req dto.CreateCommandTaskRequest, effectiveRisk string, policy *domain.ApprovalPolicy) bool {
	if req.Emergency {
		if s.rbacService != nil {
			has, _ := s.rbacService.HasPermission(ctx, userID, "command:emergency")
			return has
		}
		return false
	}
	if s.rbacService != nil {
		has, _ := s.rbacService.HasPermission(ctx, userID, "command:bypass_approval")
		if has {
			return true
		}
	}
	return false
}

func (s *Service) resolveApprover(ctx context.Context, policy *domain.ApprovalPolicy) (*string, *string) {
	if policy == nil {
		return nil, nil
	}
	return policy.ApproverUserID, policy.ApproverRoleID
}

func (s *Service) ApproveTask(ctx context.Context, tenantID, taskID, approverUserID, note string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return domain.ErrCommandTaskNotFound
	}
	if task.TenantID != tenantID {
		return domain.ErrCommandTaskNotFound
	}
	if task.Status != domain.CommandTaskPendingApproval {
		return domain.ErrCommandTaskInvalidState
	}
	if time.Now().After(task.ExpiresAt) {
		task.Status = domain.CommandTaskExpired
		s.taskRepo.Update(ctx, task)
		return domain.ErrCommandTaskExpired
	}
	if task.CreatedByUserID == approverUserID {
		policy, _ := s.policyService.FindPolicy(ctx, tenantID, task.EffectiveRisk)
		if policy != nil && policy.SeparationOfDuty {
			return domain.ErrSeparationOfDutyViolation
		}
	}

	task.Status = domain.CommandTaskApproved
	task.ApprovedByID = &approverUserID
	task.ApprovalNote = note
	now := time.Now()
	task.ApprovedAt = &now

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return err
	}

	s.recordAudit(ctx, tenantID, task.ID, approverUserID, "command.task_approved", map[string]interface{}{
		"note": note,
	})

	go s.dispatchToDevices(context.Background(), task)
	return nil
}

func (s *Service) DenyTask(ctx context.Context, tenantID, taskID, approverUserID, reason string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return domain.ErrCommandTaskNotFound
	}
	if task.TenantID != tenantID {
		return domain.ErrCommandTaskNotFound
	}
	if task.Status != domain.CommandTaskPendingApproval {
		return domain.ErrCommandTaskInvalidState
	}

	task.Status = domain.CommandTaskDenied
	task.ApprovedByID = &approverUserID
	task.ApprovalNote = reason

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return err
	}

	s.recordAudit(ctx, tenantID, task.ID, approverUserID, "command.task_denied", map[string]interface{}{
		"reason": reason,
	})

	return nil
}

func (s *Service) CancelTask(ctx context.Context, tenantID, taskID, userID string) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return domain.ErrCommandTaskNotFound
	}
	if task.TenantID != tenantID {
		return domain.ErrCommandTaskNotFound
	}
	if task.Status != domain.CommandTaskPendingApproval {
		return domain.ErrCommandTaskInvalidState
	}
	if task.CreatedByUserID != userID {
		return domain.ErrInsufficientPermission
	}

	task.Status = domain.CommandTaskCancelled
	return s.taskRepo.Update(ctx, task)
}

func (s *Service) GetTask(ctx context.Context, tenantID, taskID string) (*dto.CommandTaskResponse, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil || task.TenantID != tenantID {
		return nil, domain.ErrCommandTaskNotFound
	}
	return s.taskToResponse(ctx, task)
}

func (s *Service) ListTasks(ctx context.Context, tenantID string, filters repository.CommandTaskFilters, limit, offset int) (*dto.CommandTaskListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	tasks, total, err := s.taskRepo.ListByTenant(ctx, tenantID, filters, limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CommandTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp, err := s.taskToResponse(ctx, t)
		if err != nil {
			continue
		}
		items = append(items, *resp)
	}
	return &dto.CommandTaskListResponse{Tasks: items, Total: total}, nil
}

func (s *Service) ListPendingApprovals(ctx context.Context, tenantID, approverUserID string) ([]dto.CommandTaskResponse, error) {
	tasks, err := s.taskRepo.ListPendingByApprover(ctx, tenantID, approverUserID)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CommandTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		resp, err := s.taskToResponse(ctx, t)
		if err != nil {
			continue
		}
		items = append(items, *resp)
	}
	return items, nil
}

func (s *Service) CountPendingApprovals(ctx context.Context, tenantID, approverUserID string) (int64, error) {
	return s.taskRepo.CountPendingByApprover(ctx, tenantID, approverUserID)
}

func (s *Service) HandleExecutionResult(ctx context.Context, executionID string, status string, output *string, exitCode *int, durationMs *int) error {
	exec, err := s.execRepo.GetByID(ctx, executionID)
	if err != nil || exec == nil {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	exec.Status = domain.CommandExecutionStatus(status)
	exec.Output = output
	exec.ExitCode = exitCode
	exec.DurationMs = durationMs
	now := time.Now()
	exec.FinishedAt = &now

	if err := s.execRepo.Update(ctx, exec); err != nil {
		return err
	}

	s.updateTaskStatus(ctx, exec.TaskID)
	return nil
}

func (s *Service) updateTaskStatus(ctx context.Context, taskID string) {
	execs, err := s.execRepo.ListByTaskID(ctx, taskID)
	if err != nil {
		return
	}

	allDone := true
	anyFailed := false
	anySucceeded := false
	for _, e := range execs {
		switch e.Status {
		case domain.ExecutionSucceeded:
			anySucceeded = true
		case domain.ExecutionFailed, domain.ExecutionTimeout:
			anyFailed = true
		case domain.ExecutionSkipped:
			// skip
		default:
			allDone = false
		}
	}

	if !allDone {
		task, _ := s.taskRepo.GetByID(ctx, taskID)
		if task != nil && task.Status == domain.CommandTaskApproved {
			task.Status = domain.CommandTaskExecuting
			s.taskRepo.Update(ctx, task)
		}
		return
	}

	task, _ := s.taskRepo.GetByID(ctx, taskID)
	if task == nil {
		return
	}

	now := time.Now()
	task.CompletedAt = &now

	if anyFailed && anySucceeded {
		task.Status = domain.CommandTaskPartialDone
	} else if anyFailed {
		task.Status = domain.CommandTaskFailed
	} else {
		task.Status = domain.CommandTaskCompleted
	}

	s.taskRepo.Update(ctx, task)
}

func (s *Service) dispatchToDevices(ctx context.Context, task *domain.CommandTask) {
	deviceIDs := task.ParseDeviceIDs()
	for _, deviceID := range deviceIDs {
		exec, _ := s.execRepo.GetByTaskAndDevice(ctx, task.ID, deviceID)
		if exec == nil {
			continue
		}
		exec.Status = domain.ExecutionSent
		now := time.Now()
		exec.SentAt = &now
		s.execRepo.Update(ctx, exec)

		if s.gatewayClient != nil {
			evt := infrastructure.SessionEvent{
				EventID:   fmt.Sprintf("cmd_%s", exec.ID),
				EventType: "command.execute",
				TenantID:  task.TenantID,
				DeviceID:  deviceID,
				SessionID: task.ID,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Payload: map[string]interface{}{
					"task_id":         task.ID,
					"execution_id":    exec.ID,
					"command_type":    task.CommandType,
					"command_payload": task.CommandPayload,
				},
			}
			if err := s.gatewayClient.SendToDevice(ctx, deviceID, evt); err != nil {
				slog.Error("[command] Failed to dispatch to device", "device_id", deviceID, "error", err)
				exec.Status = domain.ExecutionSkipped
				errMsg := err.Error()
				exec.ErrorMessage = &errMsg
				s.execRepo.Update(ctx, exec)
			}
		}
	}
}

func (s *Service) recordAudit(ctx context.Context, tenantID, taskID, actorID, eventType string, data map[string]interface{}) {
	if s.auditRepo == nil {
		return
	}
	data["task_id"] = taskID
	payloadJSON, _ := json.Marshal(data)
	event := &domain.AuditEvent{
		ID:               ulid.Make().String(),
		TenantID:         tenantID,
		SessionID:        &taskID,
		EventType:        eventType,
		EventPayloadJSON: string(payloadJSON),
	}
	if err := s.auditRepo.Create(ctx, event); err != nil {
		slog.Error("[command] Failed to record audit", "event_type", eventType, "error", err)
	}
}

func (s *Service) taskToResponse(ctx context.Context, task *domain.CommandTask) (*dto.CommandTaskResponse, error) {
	resp := &dto.CommandTaskResponse{
		ID:             task.ID,
		TenantID:       task.TenantID,
		CreatedBy:      task.CreatedByUserID,
		ApproverID:     task.ApproverUserID,
		ApprovedBy:     task.ApprovedByID,
		Title:          task.Title,
		CommandType:    task.CommandType,
		CommandPayload: task.CommandPayload,
		DeviceIDs:      task.ParseDeviceIDs(),
		RiskLevel:      task.RiskLevel,
		EffectiveRisk:  task.EffectiveRisk,
		BypassApproval: task.BypassApproval,
		Emergency:      task.Emergency,
		TargetEnv:      task.TargetEnv,
		ChangeTicket:   task.ChangeTicket,
		BusinessApp:    task.BusinessApp,
		Note:           task.Note,
		Status:         string(task.Status),
		ApprovalNote:   task.ApprovalNote,
		ExpiresAt:      task.ExpiresAt,
		ApprovedAt:     task.ApprovedAt,
		CompletedAt:    task.CompletedAt,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
	}

	execs, _ := s.execRepo.ListByTaskID(ctx, task.ID)
	for _, e := range execs {
		resp.Executions = append(resp.Executions, dto.CommandExecutionResponse{
			ID:         e.ID,
			TaskID:     e.TaskID,
			DeviceID:   e.DeviceID,
			Status:     string(e.Status),
			Output:     e.Output,
			Error:      e.ErrorMessage,
			ExitCode:   e.ExitCode,
			DurationMs: e.DurationMs,
			SentAt:     e.SentAt,
			StartedAt:  e.StartedAt,
			FinishedAt: e.FinishedAt,
			CreatedAt:  e.CreatedAt,
		})
	}
	return resp, nil
}
