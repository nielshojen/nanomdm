package mongodb

import (
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/storage"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoDB) StoreBootstrapToken(r *mdm.Request, msg *mdm.SetBootstrapToken) error {
	if r.ParentID != "" {
		return storage.ErrDeviceChannelOnly
	}
	if err := s.updateLastSeen(r); err != nil {
		return fmt.Errorf("updating last seen: %w", err)
	}
	t := now()
	_, err := s.devices.UpdateOne(
		r.Context(),
		idFilter(r.ID),
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "bootstrap_token", Value: []byte(msg.BootstrapToken.BootstrapToken)},
			{Key: "bootstrap_token_at", Value: t},
			{Key: "updated_at", Value: t},
		}}},
	)
	return err
}

func (s *MongoDB) RetrieveBootstrapToken(r *mdm.Request, _ *mdm.GetBootstrapToken) (*mdm.BootstrapToken, error) {
	if r.ParentID != "" {
		return nil, storage.ErrDeviceChannelOnly
	}
	if err := s.updateLastSeen(r); err != nil {
		return nil, fmt.Errorf("updating last seen: %w", err)
	}
	var doc deviceDocument
	err := s.devices.FindOne(
		r.Context(),
		idFilter(r.ID),
		options.FindOne().SetProjection(bson.D{{Key: "bootstrap_token", Value: 1}}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) || len(doc.BootstrapToken) < 1 {
		return nil, nil
	}
	return &mdm.BootstrapToken{BootstrapToken: doc.BootstrapToken}, err
}
