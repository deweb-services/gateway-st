package mongo

import (
	"context"

	// migration lib requirement.
	_ "github.com/golang-migrate/migrate/v4/database/mongodb"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

func (mrs *MongoRepositorySuite) mongoContainerStart(ctx context.Context) (*mongodb.MongoDBContainer, string) {
	mongodbContainer, err := mongodb.Run(ctx, "mongo:7")
	mrs.Require().NoError(err)

	connectionString, err := mongodbContainer.ConnectionString(ctx)
	mrs.Require().NoError(err)

	return mongodbContainer, connectionString
}
