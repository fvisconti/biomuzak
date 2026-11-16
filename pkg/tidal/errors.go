package tidal

import "errors"

// Sentinel errors returned by the Tidal auth / import flow.
var (
	// ErrAuthorizationPending is returned while the user has not yet confirmed
	// the device authorization in their Tidal app.
	ErrAuthorizationPending = errors.New("authorization pending")

	// ErrAuthorizationExpired is returned when the device code has expired.
	ErrAuthorizationExpired = errors.New("authorization expired")

	// ErrSlowDown is returned when the client is polling too frequently.
	ErrSlowDown = errors.New("slow down")

	// ErrNotConnected is returned when the user has no Tidal connection.
	ErrNotConnected = errors.New("not connected to tidal")
)
