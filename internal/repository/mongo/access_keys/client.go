package access_keys

import "go.mongodb.org/mongo-driver/v2/mongo"

type Client struct {
	*mongo.Collection
}

func New(accessKeys *mongo.Collection) *Client {
	return &Client{Collection: accessKeys}
}
