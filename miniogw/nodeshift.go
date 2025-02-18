// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package miniogw

import (
	"context"
)

type (
	projectUUIDKey struct{}
	Info           struct {
		projectUUID        string
		internalBucketPath string
	}
)

// WithInfo injects project into ctx under a specific key.
func WithInfo(ctx context.Context, projectUUID, fullBucketPath string) context.Context {
	return context.WithValue(ctx, projectUUIDKey{}, Info{projectUUID: projectUUID, internalBucketPath: fullBucketPath})
}

// GetInfo retrieves required fields for requests.
func GetInfo(ctx context.Context) (Info, bool) {
	info, ok := ctx.Value(projectUUIDKey{}).(Info)
	return info, ok
}
