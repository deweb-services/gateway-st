package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"storj.io/minio/pkg/bucket/policy"
)

type AccessKey struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty"`
	ProjectUUID         string             `bson:"project_uuid"`
	BucketName          string             `bson:"bucket_name"`
	AccessKey           string             `bson:"access_key"`
	SecretID            string             `bson:"secret_id"`
	CloudflareRecordIDs []string           `bson:"cloudflare_record_ids"`
	Policy              *policy.Policy     `bson:"policy"`
	CreatedAt           time.Time          `bson:"created_at"`
}
