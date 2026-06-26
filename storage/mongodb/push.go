package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func (s *MongoDB) RetrievePushInfo(ctx context.Context, ids []string) (map[string]*mdm.Push, error) {
	if len(ids) < 1 {
		return nil, errors.New("no ids provided")
	}

	cursor, err := s.enrollments.Find(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: ids}}}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	pushInfos := make(map[string]*mdm.Push)
	for cursor.Next(ctx) {
		var doc enrollmentDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		push := &mdm.Push{
			Topic:     doc.Topic,
			PushMagic: doc.PushMagic,
		}
		if err := push.SetTokenString(doc.TokenHex); err != nil {
			return nil, fmt.Errorf("setting push token for %s: %w", doc.ID, err)
		}
		pushInfos[doc.ID] = push
	}
	return pushInfos, cursor.Err()
}
