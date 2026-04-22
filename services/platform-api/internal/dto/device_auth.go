package dto

// DeviceAuthInitResponse is the device authorization (RFC 8628) "device code" response.
type DeviceAuthInitResponse struct {
	DeviceCode string `json:"device_code"`
	UserCode   string `json:"user_code"`
	ExpiresIn  int    `json:"expires_in"` // seconds
	Interval   int    `json:"interval"`  // min polling interval in seconds
}

// DeviceAuthPollResponse is the result of polling for tokens (OAuth-style) or a pending/terminal state.
type DeviceAuthPollResponse struct {
	// Error is an OAuth2-style error code when the client should keep waiting or stop: authorization_pending, access_denied, expired_token, etc.
	// Empty when the flow completed successfully and access_token is set.
	Error string `json:"error,omitempty"`
	// ErrorDescription is a human-readable detail for the error, if any.
	ErrorDescription string `json:"error_description,omitempty"`

	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	// IdeClientTokenID is the stored token record (opaque id); useful for introspection and revocation.
	IdeClientTokenID string `json:"ide_client_token_id,omitempty"`
}

// DeviceAuthInitAPIRequest is the optional body for starting a device authorization session.
type DeviceAuthInitAPIRequest struct {
	DeviceInfo string `json:"device_info"`
}

// DeviceAuthPollAPIRequest is the device client poll body.
type DeviceAuthPollAPIRequest struct {
	DeviceCode string `json:"device_code" binding:"required"`
}

// DeviceAuthConfirmRequest is submitted by a logged-in console user (identity from JWT).
type DeviceAuthConfirmRequest struct {
	UserCode string `json:"user_code" binding:"required"`
	Approve  bool   `json:"approve"`
}

// IdeClientRefreshRequest exchanges a refresh token for a new access/refresh pair.
type IdeClientRefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// IdeClientTokenResponse is returned on successful device-token poll and refresh; plaintext tokens are only present in these responses.
type IdeClientTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	TokenID      string `json:"token_id"`
}
