package miniogw

import (
	"context"
	"fmt"

	"storj.io/minio/cmd"
	"storj.io/minio/pkg/bucket/policy"

	"github.com/deweb-services/gateway-st/internal/domain"
)

// SetBucketPolicy sets policy on bucket.
func (layer *gatewayLayer) SetBucketPolicy(ctx context.Context, bucket string, bucketPolicy *policy.Policy) error {
	info, found := GetInfo(ctx)
	if !found {
		return fmt.Errorf("failed to find project uuid value: %w", domain.ErrNotFound)
	}

	v, err := layer.accessKeys.Create(ctx, info.projectUUID, bucket)
	if err != nil {
		return fmt.Errorf("failed to create access keys: %w", err)
	}

	ids, err := layer.cloudflare.Create(ctx, bucket, info.internalBucketPath, v.AccessKey)
	if err != nil {
		return fmt.Errorf("failed to create cloudflare dns records: %w", err)
	}

	v.CloudflareRecordIDs = ids
	v.Policy = bucketPolicy

	if _, err := layer.mongo.AccessKeyer.CreateOrUpdate(ctx, v); err != nil {
		return fmt.Errorf("failed to create access key: %w", err)
	}

	return cmd.NotImplemented{}
}

// GetBucketPolicy will get policy on bucket.
func (layer *gatewayLayer) GetBucketPolicy(ctx context.Context, bucket string) (*policy.Policy, error) {
	info, found := GetInfo(ctx)
	if !found {
		return nil, fmt.Errorf("failed to find project uuid value: %w", domain.ErrNotFound)
	}

	v, err := layer.mongo.AccessKeyer.Get(ctx, info.projectUUID, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get access key: %w", err)
	}

	return v.Policy, nil
}

// DeleteBucketPolicy deletes all policies on bucket.
func (layer *gatewayLayer) DeleteBucketPolicy(ctx context.Context, bucket string) error {
	info, found := GetInfo(ctx)
	if !found {
		return fmt.Errorf("failed to find project uuid value: %w", domain.ErrNotFound)
	}

	v, err := layer.mongo.AccessKeyer.Get(ctx, info.projectUUID, bucket)
	if err != nil {
		return fmt.Errorf("failed to get access key: %w", err)
	}

	if err := layer.cloudflare.Delete(ctx, v.CloudflareRecordIDs); err != nil {
		return fmt.Errorf("failed to delete dns zone: %w", err)
	}

	if err := layer.accessKeys.Revoke(ctx, info.projectUUID, v.AccessKey, v.SecretID); err != nil {
		return fmt.Errorf("failed to revoke access key: %w", err)
	}

	if err := layer.mongo.AccessKeyer.Delete(ctx, v.ID, true); err != nil {
		return fmt.Errorf("failed to delete access key: %w", err)
	}

	return nil
}
