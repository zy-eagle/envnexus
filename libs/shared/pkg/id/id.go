package id

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// Prefixed entity ID prefixes as defined in the proposal.
const (
	PrefixTenant   = "enx_ten_"
	PrefixDevice   = "enx_dev_"
	PrefixSession  = "enx_sess_"
	PrefixApproval = "enx_apr_"
	PrefixAudit    = "enx_aud_"
	PrefixUser     = "enx_usr_"
	PrefixRole     = "enx_role_"
	PrefixJob      = "enx_job_"
	PrefixWebhook  = "enx_wh_"
	PrefixPackage  = "enx_pkg_"
	PrefixEvent    = "enx_evt_"
)

// New generates a unique ID with the given prefix.
// Format: {prefix}{timestamp_hex}{random_hex} (total ~26-32 chars).
func New(prefix string) string {
	ts := time.Now().UnixMilli()
	buf := make([]byte, 6)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("%s%x%s", prefix, ts, hex.EncodeToString(buf))
}

// NewEventID generates a short event ID for WebSocket event deduplication.
func NewEventID() string {
	return New(PrefixEvent)
}
