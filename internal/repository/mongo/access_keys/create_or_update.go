package access_keys

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/deweb-services/gateway-st/internal/domain"
)

func (c *Client) CreateOrUpdate(ctx context.Context, ak *domain.AccessKey) (string, error) {
	ak.CreatedAt = time.Now()

	filter := bson.M{"project_uuid": ak.ProjectUUID, "bucket_name": ak.BucketName}
	if err := c.FindOne(ctx, filter).Err(); err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return "", fmt.Errorf("failed to find access key: %w", err)
		}

		ak.ID = primitive.NewObjectID()
	}

	data, err := bson.Marshal(ak)
	if err != nil {
		return "", fmt.Errorf("failed to marshal access key: %w", err)
	}

	if _, err := c.ReplaceOne(ctx, filter, data, options.Replace().SetUpsert(true)); err != nil {
		return "", fmt.Errorf("failed to upsert bucket: %w", err)
	}

	return ak.ID.Hex(), nil
}
