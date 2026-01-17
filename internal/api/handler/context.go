package handler

import (
	"context"

	"github.com/breatheroute/breatheroute/internal/api/middleware"
)

// GetUserID retrieves the authenticated user ID from the context.
// This is a convenience wrapper around middleware.GetUserID.
func GetUserID(ctx context.Context) string {
	return middleware.GetUserID(ctx)
}
