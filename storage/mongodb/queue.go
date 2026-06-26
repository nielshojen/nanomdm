package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoDB) EnqueueCommand(ctx context.Context, ids []string, cmd *mdm.Command) (map[string]error, error) {
	if len(ids) < 1 {
		return nil, errors.New("no id(s) supplied to queue command to")
	}

	t := now()
	_, err := s.commands.InsertOne(ctx, commandDocument{
		ID:          cmd.CommandUUID,
		RequestType: cmd.Command.RequestType,
		Command:     cmd.Raw,
		CreatedAt:   t,
		UpdatedAt:   t,
	})
	if err != nil {
		if isDuplicateKey(err) {
			return nil, fmt.Errorf("command already exists: %s", cmd.CommandUUID)
		}
		return nil, err
	}

	queueDocs := make([]queueDocument, 0, len(ids))
	for _, id := range ids {
		queueDocs = append(queueDocs, queueDocument{
			ID:           queueKey(id, cmd.CommandUUID),
			EnrollmentID: id,
			CommandUUID:  cmd.CommandUUID,
			Active:       true,
			Priority:     0,
			CreatedAt:    t,
			UpdatedAt:    t,
		})
	}
	if _, err = s.enrollmentQueue.InsertMany(ctx, queueDocs); err != nil {
		return nil, err
	}
	return map[string]error{}, nil
}

func (s *MongoDB) StoreCommandReport(r *mdm.Request, result *mdm.CommandResults) error {
	if err := s.updateLastSeen(r); err != nil {
		return err
	}
	if result.Status == "Idle" {
		return nil
	} else if result.CommandUUID == "" {
		return errors.New("empty command UUID")
	}
	if s.rm && result.Status != "NotNow" {
		return s.deleteCommand(r.Context(), r.ID, result.CommandUUID)
	}

	t := now()
	set := bson.D{
		{Key: "id", Value: r.ID},
		{Key: "command_uuid", Value: result.CommandUUID},
		{Key: "status", Value: result.Status},
		{Key: "result", Value: result.Raw},
		{Key: "updated_at", Value: t},
	}
	setOnInsert := bson.D{{Key: "created_at", Value: t}}
	update := bson.D{
		{Key: "$set", Value: set},
		{Key: "$setOnInsert", Value: setOnInsert},
	}
	if result.Status == "NotNow" {
		setOnInsert = append(setOnInsert, bson.E{Key: "not_now_at", Value: t})
		update[1].Value = setOnInsert
		update = append(update, bson.E{Key: "$inc", Value: bson.D{{Key: "not_now_tally", Value: 1}}})
	}

	_, err := s.commandResults.UpdateOne(
		r.Context(),
		idFilter(queueKey(r.ID, result.CommandUUID)),
		update,
		options.UpdateOne().SetUpsert(true),
	)
	return err
}

func (s *MongoDB) RetrieveNextCommand(r *mdm.Request, skipNotNow bool) (*mdm.Command, error) {
	cursor, err := s.enrollmentQueue.Find(
		r.Context(),
		bson.D{
			{Key: "id", Value: r.ID},
			{Key: "active", Value: true},
		},
		options.Find().SetSort(bson.D{
			{Key: "priority", Value: -1},
			{Key: "created_at", Value: 1},
		}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(r.Context())

	for cursor.Next(r.Context()) {
		var queueDoc queueDocument
		if err := cursor.Decode(&queueDoc); err != nil {
			return nil, err
		}
		usable, err := s.queueResultUsable(r.Context(), queueDoc, skipNotNow)
		if err != nil {
			return nil, err
		}
		if !usable {
			continue
		}
		var commandDoc commandDocument
		err = s.commands.FindOne(r.Context(), idFilter(queueDoc.CommandUUID)).Decode(&commandDoc)
		if errors.Is(err, mongo.ErrNoDocuments) {
			continue
		} else if err != nil {
			return nil, err
		}
		return &mdm.Command{
			CommandUUID: commandDoc.ID,
			Command: struct {
				RequestType string
			}{
				RequestType: commandDoc.RequestType,
			},
			Raw: commandDoc.Command,
		}, nil
	}
	return nil, cursor.Err()
}

func (s *MongoDB) ClearQueue(r *mdm.Request) error {
	if r.ParentID != "" {
		return errors.New("can only clear a device channel queue")
	}

	enrollmentIDs, err := s.enrollmentIDsForDevice(r.Context(), r.ID)
	if err != nil {
		return err
	}

	var clearIDs []string
	for _, enrollmentID := range enrollmentIDs {
		cursor, err := s.enrollmentQueue.Find(
			r.Context(),
			bson.D{
				{Key: "id", Value: enrollmentID},
				{Key: "active", Value: true},
			},
		)
		if err != nil {
			return err
		}
		for cursor.Next(r.Context()) {
			var queueDoc queueDocument
			if err := cursor.Decode(&queueDoc); err != nil {
				cursor.Close(r.Context())
				return err
			}
			usable, err := s.queueResultUsable(r.Context(), queueDoc, false)
			if err != nil {
				cursor.Close(r.Context())
				return err
			}
			if usable {
				clearIDs = append(clearIDs, queueDoc.ID)
			}
		}
		if err := cursor.Close(r.Context()); err != nil {
			return err
		}
	}
	if len(clearIDs) < 1 {
		return nil
	}

	t := now()
	_, err = s.enrollmentQueue.UpdateMany(
		r.Context(),
		bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: clearIDs}}}},
		bson.D{{Key: "$set", Value: bson.D{
			{Key: "active", Value: false},
			{Key: "updated_at", Value: t},
		}}},
	)
	return err
}

func (s *MongoDB) queueResultUsable(ctx context.Context, queueDoc queueDocument, skipNotNow bool) (bool, error) {
	var result commandResultDocument
	err := s.commandResults.FindOne(ctx, idFilter(queueDoc.ID)).Decode(&result)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return true, nil
	} else if err != nil {
		return false, err
	}
	return result.Status == "NotNow" && !skipNotNow, nil
}

func (s *MongoDB) deleteCommand(ctx context.Context, id, uuid string) error {
	if _, err := s.enrollmentQueue.DeleteOne(ctx, idFilter(queueKey(id, uuid))); err != nil {
		return err
	}
	if _, err := s.commandResults.DeleteOne(ctx, idFilter(queueKey(id, uuid))); err != nil {
		return err
	}

	queueCount, err := s.enrollmentQueue.CountDocuments(ctx, bson.D{{Key: "command_uuid", Value: uuid}}, options.Count().SetLimit(1))
	if err != nil {
		return err
	}
	if queueCount > 0 {
		return nil
	}
	resultCount, err := s.commandResults.CountDocuments(ctx, bson.D{{Key: "command_uuid", Value: uuid}}, options.Count().SetLimit(1))
	if err != nil {
		return err
	}
	if resultCount > 0 {
		return nil
	}
	_, err = s.commands.DeleteOne(ctx, idFilter(uuid))
	return err
}

func (s *MongoDB) enrollmentIDsForDevice(ctx context.Context, deviceID string) ([]string, error) {
	cursor, err := s.enrollments.Find(
		ctx,
		bson.D{{Key: "device_id", Value: deviceID}},
		options.Find().SetProjection(bson.D{{Key: "_id", Value: 1}}),
	)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var doc enrollmentDocument
		if err := cursor.Decode(&doc); err != nil {
			return nil, err
		}
		ids = append(ids, doc.ID)
	}
	return ids, cursor.Err()
}

func queueKey(id, commandUUID string) string {
	return compositeKey(id, commandUUID)
}
