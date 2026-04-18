package domain

import "time"

type FileAccessStatus string

const (
	FileAccessPending  FileAccessStatus = "pending"
	FileAccessApproved FileAccessStatus = "approved"
	FileAccessDenied   FileAccessStatus = "denied"
	FileAccessExpired  FileAccessStatus = "expired"
)

type FileAccessAction string

const (
	FileAccessBrowse   FileAccessAction = "browse"
	FileAccessPreview  FileAccessAction = "preview"
	FileAccessDownload FileAccessAction = "download"
)

type FileAccessRequest struct {
	ID               string           `json:"id"                gorm:"primaryKey;size:26"`
	TenantID         string           `json:"tenant_id"         gorm:"size:26;not null;index"`
	DeviceID         string           `json:"device_id"         gorm:"size:26;not null;index"`
	RequestedBy      string           `json:"requested_by"      gorm:"size:26;not null"`
	ApprovedBy       *string          `json:"approved_by"       gorm:"size:26"`
	ApproverUserID   *string          `json:"approver_user_id"  gorm:"size:26"`
	ApproverRoleID   *string          `json:"approver_role_id"  gorm:"size:26"`
	PolicySnapshotID *string          `json:"policy_snapshot_id" gorm:"size:26"`
	Path             string           `json:"path"              gorm:"size:1024;not null"`
	Action           FileAccessAction `json:"action"            gorm:"size:32;not null"`
	Status           FileAccessStatus `json:"status"            gorm:"size:32;not null"`
	ResultJSON       string           `json:"result"            gorm:"type:text"`
	Note             string           `json:"note"              gorm:"type:text"`
	ExpiresAt        time.Time        `json:"expires_at"        gorm:"not null"`
	ResolvedAt       *time.Time       `json:"resolved_at"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

func (FileAccessRequest) TableName() string { return "file_access_requests" }
