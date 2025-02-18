package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"storj.io/minio/pkg/bucket/policy"

	"github.com/deweb-services/gateway-st/internal/domain"
)

func (mrs *MongoRepositorySuite) TestAccessKeysCreate() {
	ctx := context.Background()

	for _, tc := range []struct {
		name string

		entry *domain.AccessKey

		setup  func()
		assert func(string, error)
	}{
		{
			name: "no records; expect ok",
			entry: &domain.AccessKey{
				BucketName:          "create_bucket",
				AccessKey:           "create_access_key",
				SecretID:            "create_secret_key",
				CloudflareRecordIDs: []string{},
				Policy: &policy.Policy{
					ID:         "create_bucket_policy",
					Version:    "create_bucket_policy_version",
					Statements: []policy.Statement{},
				},
				ProjectUUID: "create_project_uuid",
			},
			assert: func(objectID string, err error) {
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(objectID)

				actual, err := mrs.repo.AccessKeyer.Get(ctx, "create_project_uuid", "create_bucket")
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(actual)

				hid, err := primitive.ObjectIDFromHex(objectID)
				mrs.Require().NoError(err)

				now := time.Now()

				expected := &domain.AccessKey{
					ID:                  hid,
					BucketName:          "create_bucket",
					AccessKey:           "create_access_key",
					SecretID:            "create_secret_key",
					CloudflareRecordIDs: []string{},
					Policy: &policy.Policy{
						ID:         "create_bucket_policy",
						Version:    "create_bucket_policy_version",
						Statements: []policy.Statement{},
					},
					ProjectUUID: "create_project_uuid",
					CreatedAt:   time.Now(),
				}

				expected.CreatedAt = now
				actual.CreatedAt = now

				mrs.Assert().Equal(expected, actual)
			},
		},
		{
			name: "with record; expect ok",
			setup: func() {
				id, err := mrs.repo.AccessKeyer.CreateOrUpdate(ctx, &domain.AccessKey{
					ProjectUUID:         "update_project_uuid",
					BucketName:          "update_bucket",
					AccessKey:           "update_access_key",
					SecretID:            "update_secret_key",
					CloudflareRecordIDs: []string{"update_record_id_1", "update_record_id_2"},
					Policy: &policy.Policy{
						ID:         "update_bucket_policy",
						Version:    "update_bucket_policy_version",
						Statements: []policy.Statement{},
					},
				})

				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(id)
			},
			entry: &domain.AccessKey{
				ProjectUUID:         "update_project_uuid",
				BucketName:          "update_bucket",
				AccessKey:           "update_access_key",
				SecretID:            "update_secret_key",
				CloudflareRecordIDs: []string{"update_record_id_1", "update_record_id_2", "update_record_id_3"},
				Policy: &policy.Policy{
					ID:      "update_bucket_policy",
					Version: "update_bucket_policy_version",
					Statements: []policy.Statement{
						{
							SID:    "policy_sid",
							Effect: policy.Allow,
						},
					},
				},
			},
			assert: func(objectID string, err error) {
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(objectID)

				actual, err := mrs.repo.AccessKeyer.Get(ctx, "update_project_uuid", "update_bucket")
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(actual)

				now := time.Now()

				expected := &domain.AccessKey{
					ID:                  actual.ID,
					ProjectUUID:         "update_project_uuid",
					BucketName:          "update_bucket",
					AccessKey:           "update_access_key",
					SecretID:            "update_secret_key",
					CloudflareRecordIDs: []string{"update_record_id_1", "update_record_id_2", "update_record_id_3"},
					Policy: &policy.Policy{
						ID:      "update_bucket_policy",
						Version: "update_bucket_policy_version",
						Statements: []policy.Statement{
							{
								SID:    "policy_sid",
								Effect: policy.Allow,
							},
						},
					},
				}

				expected.CreatedAt = now
				actual.CreatedAt = now

				mrs.Assert().Equal(expected, actual)
			},
		},
	} {
		mrs.Run(tc.name, func() {
			if tc.setup != nil {
				tc.setup()
			}

			tc.assert(mrs.repo.AccessKeyer.CreateOrUpdate(ctx, tc.entry))
		})
	}
}

func (mrs *MongoRepositorySuite) TestAccessKeysDelete() {
	ctx := context.Background()

	for _, tc := range []struct {
		name string

		id    primitive.ObjectID
		force bool

		setup  func() primitive.ObjectID
		assert func(error)
	}{
		{
			name: "no records; expect error",
			id:   primitive.NewObjectID(),
			assert: func(err error) {
				mrs.Require().Error(err)

				mrs.Assert().ErrorContains(err, "not found")
			},
		},
		{
			name: "with existing record; expect ok",
			setup: func() primitive.ObjectID {
				id, err := mrs.repo.AccessKeyer.CreateOrUpdate(ctx, &domain.AccessKey{
					BucketName:  "delete_bucket",
					ProjectUUID: "delete_project_uuid",
				})
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(id)

				hid, err := primitive.ObjectIDFromHex(id)
				mrs.Require().NoError(err)

				return hid
			},
			assert: func(err error) {
				mrs.Require().NoError(err)

				actual, err := mrs.repo.AccessKeyer.Get(ctx, "delete_project_uuid", "delete_bucket")
				mrs.Require().Error(err)
				mrs.Require().Empty(actual)

				mrs.Assert().ErrorContains(err, "not found")
			},
		},
		{
			name: "with two records; expect deleted one",
			setup: func() primitive.ObjectID {
				id, err := mrs.repo.AccessKeyer.CreateOrUpdate(ctx, &domain.AccessKey{
					BucketName:  "delete_bucket",
					ProjectUUID: "delete_project_uuid",
				})
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(id)

				id2, err := mrs.repo.AccessKeyer.CreateOrUpdate(ctx, &domain.AccessKey{
					BucketName:  "not_delete_bucket",
					ProjectUUID: "delete_project_uuid",
				})
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(id2)

				hid, err := primitive.ObjectIDFromHex(id)
				mrs.Require().NoError(err)

				return hid
			},
			assert: func(err error) {
				mrs.Require().NoError(err)

				actual, err := mrs.repo.AccessKeyer.Get(ctx, "delete_project_uuid", "delete_bucket")
				mrs.Require().Error(err)
				mrs.Require().Empty(actual)

				// nolint: testifylint
				mrs.Assert().ErrorContains(err, "not found")

				actual, err = mrs.repo.AccessKeyer.Get(ctx, "delete_project_uuid", "not_delete_bucket")
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(actual)
			},
		},
	} {
		mrs.Run(tc.name, func() {
			id := tc.id

			if tc.setup != nil {
				id = tc.setup()
			}

			tc.assert(mrs.repo.AccessKeyer.Delete(ctx, id, tc.force))
		})
	}
}

func (mrs *MongoRepositorySuite) TestAccessKeysGet() {
	ctx := context.Background()

	for _, tc := range []struct {
		name string

		projectUUID, bucket string

		setup  func()
		assert func(*domain.AccessKey, error)
	}{
		{
			name:        "no records; expect error",
			bucket:      "get_bucket",
			projectUUID: "get_bucket",
			assert: func(actual *domain.AccessKey, err error) {
				mrs.Require().Error(err)
				mrs.Require().Nil(actual)

				mrs.Assert().ErrorContains(err, "not found")
			},
		},
		{
			name:        "with existing record; expect ok",
			bucket:      "get_bucket",
			projectUUID: "get_project_uuid",
			setup: func() {
				b, err := mrs.repo.AccessKeyer.CreateOrUpdate(ctx, &domain.AccessKey{
					ProjectUUID:         "get_project_uuid",
					BucketName:          "get_bucket",
					AccessKey:           "get_access_key",
					SecretID:            "get_secret_key",
					CloudflareRecordIDs: []string{"get_record_id_1"},
					Policy: &policy.Policy{
						ID:         "get_bucket_policy",
						Version:    "get_bucket_policy_version",
						Statements: []policy.Statement{},
					},
				})
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(b)
			},
			assert: func(actual *domain.AccessKey, err error) {
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(actual)

				expected := &domain.AccessKey{
					ID:                  actual.ID,
					ProjectUUID:         "get_project_uuid",
					BucketName:          "get_bucket",
					AccessKey:           "get_access_key",
					SecretID:            "get_secret_key",
					CloudflareRecordIDs: []string{"get_record_id_1"},
					Policy: &policy.Policy{
						ID:         "get_bucket_policy",
						Version:    "get_bucket_policy_version",
						Statements: []policy.Statement{},
					},
					CreatedAt: actual.CreatedAt,
				}

				mrs.Assert().Equal(expected, actual)
			},
		},
		{
			name:        "with two records; expect ok",
			bucket:      "get_bucket_1",
			projectUUID: "get_project_uuid",
			setup: func() {
				b, err := mrs.repo.AccessKeyer.CreateOrUpdate(ctx, &domain.AccessKey{
					BucketName:  "get_bucket_1",
					ProjectUUID: "get_project_uuid",
				})
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(b)

				b, err = mrs.repo.AccessKeyer.CreateOrUpdate(ctx, &domain.AccessKey{
					BucketName:  "get_bucket_2",
					ProjectUUID: "get_project_uuid",
				})
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(b)
			},
			assert: func(actual *domain.AccessKey, err error) {
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(actual)

				actual, err = mrs.repo.AccessKeyer.Get(ctx, "get_project_uuid", "get_bucket_1")
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(actual)

				mrs.Assert().Equal("get_project_uuid", actual.ProjectUUID)
				mrs.Assert().Equal("get_bucket_1", actual.BucketName)

				actual, err = mrs.repo.AccessKeyer.Get(ctx, "get_project_uuid", "get_bucket_2")
				mrs.Require().NoError(err)
				mrs.Require().NotEmpty(actual)

				mrs.Assert().Equal("get_project_uuid", actual.ProjectUUID)
				mrs.Assert().Equal("get_bucket_2", actual.BucketName)
			},
		},
	} {
		mrs.Run(tc.name, func() {
			if tc.setup != nil {
				tc.setup()
			}

			tc.assert(mrs.repo.AccessKeyer.Get(ctx, tc.projectUUID, tc.bucket))
		})
	}
}
