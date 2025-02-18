package access_keys

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/deweb-services/gateway-st/internal/domain"
)

func (c *Client) Get(ctx context.Context, projectUUID, name string) (*domain.AccessKey, error) {
	be := &domain.AccessKey{}
	filter := bson.M{"bucket_name": name, "project_uuid": projectUUID}

	if err := c.FindOne(ctx, filter).Decode(be); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("failed to find access key: %w", domain.ErrNotFound)
		}

		return nil, fmt.Errorf("failed to find access key: %w", err)
	}

	return be, nil
}
