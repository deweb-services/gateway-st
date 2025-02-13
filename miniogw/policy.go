package miniogw

import (
	"context"

	"storj.io/minio/cmd"
	"storj.io/minio/pkg/bucket/policy"
)

// SetBucketPolicy sets policy on bucket.
func (layer *gatewayLayer) SetBucketPolicy(ctx context.Context, bucket string, bucketPolicy *policy.Policy) error {
	return cmd.NotImplemented{}
}

// GetBucketPolicy will get policy on bucket.
func (layer *gatewayLayer) GetBucketPolicy(ctx context.Context, bucket string) (*policy.Policy, error) {
	return nil, cmd.NotImplemented{}
}

// DeleteBucketPolicy deletes all policies on bucket.
func (layer *gatewayLayer) DeleteBucketPolicy(ctx context.Context, bucket string) error {
	return cmd.NotImplemented{}
}
