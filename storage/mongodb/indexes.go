package mongodb

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoDB) ensureIndexes(ctx context.Context) error {
	if err := s.ensureCollections(ctx); err != nil {
		return err
	}

	indexes := []struct {
		coll   *mongo.Collection
		models []mongo.IndexModel
	}{
		{
			coll: s.devices,
			models: []mongo.IndexModel{
				indexModel("serial_number", bson.D{{Key: "serial_number", Value: 1}}, options.Index().SetSparse(true)),
			},
		},
		{
			coll: s.users,
			models: []mongo.IndexModel{
				indexModel("device_id", bson.D{{Key: "device_id", Value: 1}}, nil),
			},
		},
		{
			coll: s.enrollments,
			models: []mongo.IndexModel{
				indexModel("device_id", bson.D{{Key: "device_id", Value: 1}}, nil),
				indexModel("user_id_unique", bson.D{{Key: "user_id", Value: 1}}, options.Index().SetUnique(true).SetSparse(true)),
				indexModel("type", bson.D{{Key: "type", Value: 1}}, nil),
			},
		},
		{
			coll: s.commandResults,
			models: []mongo.IndexModel{
				indexModel("status", bson.D{{Key: "status", Value: 1}}, nil),
				indexModel("command_uuid", bson.D{{Key: "command_uuid", Value: 1}}, nil),
			},
		},
		{
			coll: s.enrollmentQueue,
			models: []mongo.IndexModel{
				indexModel("id_active_priority_created_at", bson.D{
					{Key: "id", Value: 1},
					{Key: "active", Value: 1},
					{Key: "priority", Value: -1},
					{Key: "created_at", Value: 1},
				}, nil),
				indexModel("command_uuid", bson.D{{Key: "command_uuid", Value: 1}}, nil),
			},
		},
		{
			coll: s.certAuth,
			models: []mongo.IndexModel{
				indexModel("id", bson.D{{Key: "id", Value: 1}}, nil),
				indexModel("sha256", bson.D{{Key: "sha256", Value: 1}}, nil),
				indexModel("id_sha256_unique", bson.D{
					{Key: "id", Value: 1},
					{Key: "sha256", Value: 1},
				}, options.Index().SetUnique(true)),
			},
		},
	}

	for _, idx := range indexes {
		if _, err := idx.coll.Indexes().CreateMany(ctx, idx.models); err != nil {
			return fmt.Errorf("creating indexes for %s: %w", idx.coll.Name(), err)
		}
	}

	return nil
}

func (s *MongoDB) ensureCollections(ctx context.Context) error {
	for _, coll := range s.collections() {
		if err := s.db.CreateCollection(ctx, coll.Name()); err != nil && !isNamespaceExists(err) {
			return fmt.Errorf("creating collection %s: %w", coll.Name(), err)
		}
	}
	return nil
}

func indexModel(name string, keys bson.D, opts *options.IndexOptionsBuilder) mongo.IndexModel {
	if opts == nil {
		opts = options.Index()
	}
	return mongo.IndexModel{
		Keys:    keys,
		Options: opts.SetName(name),
	}
}

func isNamespaceExists(err error) bool {
	var cmdErr mongo.CommandError
	return errors.As(err, &cmdErr) && cmdErr.HasErrorCode(48)
}
