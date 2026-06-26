package mongodb

import (
	"context"
	"strings"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoDB) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	return s.exists(r.Context(), s.certAuth, bson.D{{Key: "sha256", Value: strings.ToLower(hash)}})
}

func (s *MongoDB) EnrollmentHasCertHash(r *mdm.Request, _ string) (bool, error) {
	return s.exists(r.Context(), s.certAuth, bson.D{{Key: "id", Value: r.ID}})
}

func (s *MongoDB) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	return s.exists(r.Context(), s.certAuth, bson.D{
		{Key: "id", Value: r.ID},
		{Key: "sha256", Value: strings.ToLower(hash)},
	})
}

func (s *MongoDB) AssociateCertHash(r *mdm.Request, hash string) error {
	hash = strings.ToLower(hash)
	t := now()
	_, err := s.certAuth.UpdateOne(
		r.Context(),
		idFilter(certAuthKey(r.ID, hash)),
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "id", Value: r.ID},
				{Key: "sha256", Value: hash},
				{Key: "updated_at", Value: t},
			}},
			{Key: "$setOnInsert", Value: bson.D{{Key: "created_at", Value: t}}},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *MongoDB) EnrollmentFromHash(ctx context.Context, hash string) (string, error) {
	var doc certAuthDocument
	err := s.certAuth.FindOne(
		ctx,
		bson.D{{Key: "sha256", Value: strings.ToLower(hash)}},
		options.FindOne().SetProjection(bson.D{{Key: "id", Value: 1}}),
	).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return "", nil
	}
	return doc.EnrollmentID, err
}

func (s *MongoDB) exists(ctx context.Context, coll *mongo.Collection, filter bson.D) (bool, error) {
	count, err := coll.CountDocuments(ctx, filter, options.Count().SetLimit(1))
	return count > 0, err
}

func certAuthKey(id, hash string) string {
	return compositeKey(id, hash)
}
