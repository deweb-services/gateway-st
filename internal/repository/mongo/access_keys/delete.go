package access_keys

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"github.com/deweb-services/gateway-st/internal/domain"
)

func (c *Client) Delete(ctx context.Context, id primitive.ObjectID, force bool) error {
	if err := c.FindOneAndDelete(ctx, bson.M{"_id": id}).Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("failed to delete access key: %w", domain.ErrNotFound)
		}

		return fmt.Errorf("failed to delete access key: %w", err)
	}

	return nil
}
