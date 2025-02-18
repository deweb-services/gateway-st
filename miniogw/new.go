package miniogw

import (
	"context"
	"fmt"
	"net/http"

	"storj.io/minio/cmd"

	"github.com/deweb-services/gateway-st/internal/cloudflare"
	"github.com/deweb-services/gateway-st/internal/domain"
	nb "github.com/deweb-services/gateway-st/internal/node_backend"
	"github.com/deweb-services/gateway-st/internal/repository/mongo"
)

//go:generate mockgen -source new.go -destination=./mocks/mocks.go -package=mocks

type (
	accessKeyer interface {
		Create(ctx context.Context, projectUUID, bucketName string) (*domain.AccessKey, error)
		Revoke(ctx context.Context, projectUUID, accessKey, secretID string) error
	}
	cloudflarer interface {
		Create(ctx context.Context, bucket, fullBucketPath, accessKey string) ([]string, error)
		Delete(ctx context.Context, ids []string) error
	}
)

type gatewayLayer struct {
	logger debugLogger
	cmd.GatewayUnsupported
	compatibilityConfig S3CompatibilityConfig

	accessKeys accessKeyer
	cloudflare cloudflarer
	mongo      *mongo.Client
	ZoneID     string
}

type debugLogger interface {
	Info(args ...interface{})
	Infof(format string, args ...interface{})
}

type Options struct {
	Cloudflare struct {
		APIToken string
		ZoneID   string
	}
	Node struct {
		Host  string
		Token string
	}
	Mongo struct {
		Connection string
	}
}

// NewGatewayLayer implements cmd.Gateway.
func (gateway *Gateway) NewGatewayLayer(logger debugLogger, opts Options) (cmd.ObjectLayer, error) {
	cc := cloudflare.New(opts.Cloudflare.APIToken, opts.Cloudflare.ZoneID)

	ndc := nb.New(http.DefaultClient, opts.Node.Host, opts.Node.Token)

	client, err := mongo.New(context.Background(), opts.Mongo.Connection)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	return &gatewayLayer{
		logger:              logger,
		compatibilityConfig: gateway.compatibilityConfig,

		cloudflare: cc,
		accessKeys: ndc,
		mongo:      client,
		ZoneID:     opts.Cloudflare.ZoneID,
	}, nil
}
