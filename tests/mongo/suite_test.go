package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	mongo_repo "github.com/deweb-services/gateway-st/internal/repository/mongo"
)

type MongoRepositorySuite struct {
	suite.Suite

	repo           *mongo_repo.Client
	mongoDBClient  *mongo.Client
	mongoContainer *mongodb.MongoDBContainer
}

func TestMongoRepository(t *testing.T) {
	suite.Run(t, new(MongoRepositorySuite))
}

func (mrs *MongoRepositorySuite) SetupSuite() {
	ctx := context.Background()

	mc, ma := mrs.mongoContainerStart(ctx)

	mdbc, err := mongo.Connect(options.Client().ApplyURI(ma))
	mrs.Require().NoError(err)

	repo, err := mongo_repo.New(ctx, ma)
	mrs.Require().NoError(err)

	mrs.repo = repo
	mrs.mongoDBClient = mdbc
	mrs.mongoContainer = mc
}

func (mrs *MongoRepositorySuite) TearDownSuite() {
	testcontainers.CleanupContainer(mrs.T(), mrs.mongoContainer)
}

func (mrs *MongoRepositorySuite) TearDownSubTest() {
	ctx := context.Background()

	dbs, err := mrs.mongoDBClient.ListDatabaseNames(ctx, bson.M{"name": bson.M{"$nin": []string{"admin", "config", "local"}}})
	mrs.Require().NoError(err)

	for _, db := range dbs {
		err := mrs.mongoDBClient.Database(db).Drop(ctx)
		mrs.Require().NoError(err)
	}
}
