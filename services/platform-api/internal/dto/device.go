package dto

import "time"

type DeviceResponse struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	Hostname  string    `json:"hostname"`
	OSType    string    `json:"os_type"`
	Status    string    `json:"status"`
	LastSeen  time.Time `json:"last_seen_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UpdateDeviceRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}
