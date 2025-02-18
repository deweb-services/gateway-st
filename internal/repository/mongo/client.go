package mongo

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/deweb-services/gateway-st/internal/repository"
	"github.com/deweb-services/gateway-st/internal/repository/mongo/access_keys"
)

type Client struct {
	mc *mongo.Client

	AccessKeyer repository.AccessKeyer
}

func New(ctx context.Context, connectionString string) (*Client, error) {
	mc, err := mongo.Connect(options.Client().ApplyURI(connectionString))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	if err := mc.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping mongodb: %w", err)
	}

	return &Client{
		mc:          mc,
		AccessKeyer: access_keys.New(mc.Database("storage").Collection("access_keys")),
	}, nil
}

func (c *Client) Disconnect(ctx context.Context) error {
	if err := c.mc.Disconnect(ctx); err != nil {
		return fmt.Errorf("failed to disconnect: %w", err)
	}

	return nil
}
