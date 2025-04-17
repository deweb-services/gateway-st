// Package miniogw worker calculating bucket size asynchronously.
// each list request will trigger recalculating request
// cache will store project uuids to map already calculated buckets
// cache have ttl to recalculate size
package miniogw

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/sirupsen/logrus"
	"storj.io/uplink"

	nodeshift "github.com/deweb-services/go-nodeshift/project"
)

type Client struct {
	ttl    time.Duration
	bucket string

	cache *expirable.LRU[string, struct{}]
	// context has all info about storj project and project uuid
	ctxs chan context.Context
}

func NewWorker(bucket string) *Client {
	return &Client{
		bucket: bucket,
		ttl:    time.Hour,
		cache:  expirable.NewLRU[string, struct{}](1000, nil, 3*time.Hour),
		ctxs:   make(chan context.Context, 1000),
	}
}

func (c *Client) Add(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context error: %w", err)
	}

	go func() {
		c.ctxs <- context.WithoutCancel(ctx)
	}()

	return nil
}

func (c *Client) Run() {
	for ctx := range c.ctxs {
		projectUUID, found := nodeshift.GetProjectUUID(ctx)
		if !found {
			logrus.Errorf("project uuid not found")
			continue
		}

		project, found := GetUplinkProject(ctx)
		if !found {
			logrus.Errorf("uplink project not found")
			continue
		}

		if _, found := c.cache.Get(projectUUID); found {
			logrus.Infof("project uuid %s already exists", projectUUID)
			continue
		}

		if err := c.calculateBucketSizes(ctx, project, projectUUID); err != nil {
			logrus.Errorf("failed to calculate bucket sizes': %v", err)
			continue
		}

		c.cache.Add(projectUUID, struct{}{})
	}
}

func (c *Client) calculateBucketSizes(ctx context.Context, project *uplink.Project, projectUUID string) error {
	// list buckets
	buckets := project.ListObjects(ctx, c.bucket, &uplink.ListObjectsOptions{Prefix: projectUUID + "/"})

	for buckets.Next() {
		prefix := buckets.Item().Key

		// list bucket objects
		objects := project.ListObjects(ctx, c.bucket, &uplink.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
			System:    true,
		})

		name := strings.TrimSuffix(prefix, "/")
		// get bucket info for storing size
		object, err := project.StatObject(ctx, c.bucket, name)
		if err != nil {
			return fmt.Errorf("failed to stat object: %w", err)
		}

		// calculate bucket size
		size := int64(0)
		for objects.Next() {
			size += objects.Item().System.ContentLength
		}
		object.Custom["size"] = strconv.FormatInt(size, 10)

		// update bucket metadata with current size
		if err := project.UpdateObjectMetadata(ctx, c.bucket, name, object.Custom, nil); err != nil {
			return fmt.Errorf("failed to update object metadata: %w", err)
		}
	}

	return nil
}
