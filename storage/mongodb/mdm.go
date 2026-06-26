package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/cryptoutil"
	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoDB) StoreAuthenticate(r *mdm.Request, msg *mdm.Authenticate) error {
	t := now()
	set := bson.D{
		{Key: "authenticate", Value: msg.Raw},
		{Key: "authenticate_at", Value: t},
		{Key: "updated_at", Value: t},
	}
	unset := bson.D{
		{Key: "bootstrap_token", Value: ""},
		{Key: "bootstrap_token_at", Value: ""},
	}
	if r.Certificate != nil {
		set = append(set, bson.E{Key: "identity_cert", Value: cryptoutil.PEMCertificate(r.Certificate.Raw)})
	} else {
		unset = append(unset, bson.E{Key: "identity_cert", Value: ""})
	}
	if msg.SerialNumber != "" {
		set = append(set, bson.E{Key: "serial_number", Value: msg.SerialNumber})
	} else {
		unset = append(unset, bson.E{Key: "serial_number", Value: ""})
	}

	_, err := s.devices.UpdateOne(
		r.Context(),
		idFilter(r.ID),
		bson.D{
			{Key: "$set", Value: set},
			{Key: "$unset", Value: unset},
			{Key: "$setOnInsert", Value: bson.D{{Key: "created_at", Value: t}}},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *MongoDB) storeDeviceTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	t := now()
	set := bson.D{
		{Key: "token_update", Value: msg.Raw},
		{Key: "token_update_at", Value: t},
		{Key: "updated_at", Value: t},
	}
	if len(msg.UnlockToken) > 0 {
		set = append(set,
			bson.E{Key: "unlock_token", Value: msg.UnlockToken},
			bson.E{Key: "unlock_token_at", Value: t},
		)
	}
	_, err := s.devices.UpdateOne(r.Context(), idFilter(r.ID), bson.D{{Key: "$set", Value: set}})
	return err
}

func (s *MongoDB) storeUserTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	t := now()
	set := bson.D{
		{Key: "device_id", Value: r.ParentID},
		{Key: "token_update", Value: msg.Raw},
		{Key: "token_update_at", Value: t},
		{Key: "updated_at", Value: t},
	}
	if msg.UserShortName != "" {
		set = append(set, bson.E{Key: "user_short_name", Value: msg.UserShortName})
	}
	if msg.UserLongName != "" {
		set = append(set, bson.E{Key: "user_long_name", Value: msg.UserLongName})
	}
	_, err := s.users.UpdateOne(
		r.Context(),
		idFilter(r.ID),
		bson.D{
			{Key: "$set", Value: set},
			{Key: "$setOnInsert", Value: bson.D{{Key: "created_at", Value: t}}},
		},
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *MongoDB) StoreTokenUpdate(r *mdm.Request, msg *mdm.TokenUpdate) error {
	resolved := (&msg.Enrollment).Resolved()
	if err := resolved.Validate(); err != nil {
		return err
	}

	var err error
	deviceID := r.ID
	var userID string
	if resolved.IsUserChannel {
		deviceID = r.ParentID
		userID = r.ID
		err = s.storeUserTokenUpdate(r, msg)
	} else {
		err = s.storeDeviceTokenUpdate(r, msg)
	}
	if err != nil {
		return err
	}

	t := now()
	set := bson.D{
		{Key: "device_id", Value: deviceID},
		{Key: "type", Value: r.Type.String()},
		{Key: "topic", Value: msg.Topic},
		{Key: "push_magic", Value: msg.PushMagic},
		{Key: "token_hex", Value: msg.Token.String()},
		{Key: "enabled", Value: true},
		{Key: "last_seen_at", Value: t},
		{Key: "updated_at", Value: t},
	}
	unset := bson.D{}
	if userID != "" {
		set = append(set, bson.E{Key: "user_id", Value: userID})
	} else {
		unset = append(unset, bson.E{Key: "user_id", Value: ""})
	}
	update := bson.D{
		{Key: "$set", Value: set},
		{Key: "$inc", Value: bson.D{{Key: "token_update_tally", Value: 1}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "created_at", Value: t}}},
	}
	if len(unset) > 0 {
		update = append(update, bson.E{Key: "$unset", Value: unset})
	}
	_, err = s.enrollments.UpdateOne(r.Context(), idFilter(r.ID), update, options.UpdateOne().SetUpsert(true))
	return err
}

func (s *MongoDB) RetrieveTokenUpdateTally(ctx context.Context, id string) (int, error) {
	var doc enrollmentDocument
	err := s.enrollments.FindOne(
		ctx,
		idFilter(id),
		options.FindOne().SetProjection(bson.D{{Key: "token_update_tally", Value: 1}}),
	).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return 0, nil
	}
	return doc.TokenUpdateTally, err
}

func (s *MongoDB) StoreUserAuthenticate(r *mdm.Request, msg *mdm.UserAuthenticate) error {
	t := now()
	fieldName := "user_authenticate"
	fieldAtName := "user_authenticate_at"
	if msg.DigestResponse != "" {
		fieldName = "user_authenticate_digest"
		fieldAtName = "user_authenticate_digest_at"
	}

	set := bson.D{
		{Key: "device_id", Value: r.ParentID},
		{Key: fieldName, Value: msg.Raw},
		{Key: fieldAtName, Value: t},
		{Key: "updated_at", Value: t},
	}
	if msg.UserShortName != "" {
		set = append(set, bson.E{Key: "user_short_name", Value: msg.UserShortName})
	}
	if msg.UserLongName != "" {
		set = append(set, bson.E{Key: "user_long_name", Value: msg.UserLongName})
	}
	_, err := s.users.UpdateOne(
		r.Context(),
		idFilter(r.ID),
		bson.D{
			{Key: "$set", Value: set},
			{Key: "$setOnInsert", Value: bson.D{{Key: "created_at", Value: t}}},
		},
		options.UpdateOne().SetUpsert(true),
	)
	if err != nil {
		return err
	}
	return s.updateLastSeen(r)
}

func (s *MongoDB) Disable(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only disable a device channel")
	}
	t := now()
	_, err := s.enrollments.UpdateMany(
		r.Context(),
		bson.D{
			{Key: "device_id", Value: r.ID},
			{Key: "enabled", Value: true},
		},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "enabled", Value: false},
			{Key: "token_update_tally", Value: 0},
			{Key: "last_seen_at", Value: t},
			{Key: "updated_at", Value: t},
		}}},
	)
	return err
}

func (s *MongoDB) updateLastSeen(r *mdm.Request) error {
	t := now()
	_, err := s.enrollments.UpdateOne(
		r.Context(),
		idFilter(r.ID),
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "last_seen_at", Value: t},
			{Key: "updated_at", Value: t},
		}}},
	)
	if err != nil {
		err = fmt.Errorf("updating last seen: %w", err)
	}
	return err
}

func decodeCheckin(raw []byte, out chan<- interface{}) {
	msg, err := mdm.DecodeCheckin(raw)
	if err != nil {
		out <- err
		return
	}
	out <- msg
}
