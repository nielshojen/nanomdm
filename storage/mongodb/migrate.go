package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoDB) RetrieveMigrationCheckins(ctx context.Context, out chan<- interface{}) error {
	deviceCursor, err := s.devices.Find(
		ctx,
		bson.D{},
		options.Find().SetProjection(bson.D{
			{Key: "authenticate", Value: 1},
			{Key: "token_update", Value: 1},
		}),
	)
	if err != nil {
		return err
	}
	defer deviceCursor.Close(ctx)

	for deviceCursor.Next(ctx) {
		var doc deviceDocument
		if err := deviceCursor.Decode(&doc); err != nil {
			return err
		}
		for _, raw := range [][]byte{doc.Authenticate, doc.TokenUpdate} {
			if len(raw) > 0 {
				decodeCheckin(raw, out)
			}
		}
	}
	if err = deviceCursor.Err(); err != nil {
		return err
	}

	userCursor, err := s.users.Find(
		ctx,
		bson.D{{Key: "token_update", Value: bson.D{{Key: "$exists", Value: true}}}},
		options.Find().SetProjection(bson.D{{Key: "token_update", Value: 1}}),
	)
	if err != nil {
		return err
	}
	defer userCursor.Close(ctx)

	for userCursor.Next(ctx) {
		var doc userDocument
		if err := userCursor.Decode(&doc); err != nil {
			return err
		}
		if len(doc.TokenUpdate) > 0 {
			decodeCheckin(doc.TokenUpdate, out)
		}
	}
	return userCursor.Err()
}
