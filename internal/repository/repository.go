package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/deweb-services/gateway-st/internal/domain"
)

//go:generate mockgen -source repository.go -destination=./mocks/mocks.go -package=mocks

type (
	AccessKeyer interface {
		CreateOrUpdate(ctx context.Context, ak *domain.AccessKey) (string, error)
		Get(ctx context.Context, projectUUID, name string) (*domain.AccessKey, error)
		Delete(ctx context.Context, id primitive.ObjectID, force bool) error
	}
)
