package mongodb

import (
	"context"
	"crypto/tls"
	"strconv"

	"github.com/micromdm/nanomdm/cryptoutil"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (s *MongoDB) RetrievePushCert(ctx context.Context, topic string) (*tls.Certificate, string, error) {
	var doc pushCertDocument
	err := s.pushCerts.FindOne(ctx, idFilter(topic)).Decode(&doc)
	if err != nil {
		return nil, "", err
	}
	cert, err := tls.X509KeyPair(doc.CertPEM, doc.KeyPEM)
	if err != nil {
		return nil, "", err
	}
	return &cert, strconv.Itoa(doc.StaleToken), nil
}

func (s *MongoDB) IsPushCertStale(ctx context.Context, topic, staleToken string) (bool, error) {
	staleTokenInt, err := strconv.Atoi(staleToken)
	if err != nil {
		return true, err
	}
	var doc pushCertDocument
	err = s.pushCerts.FindOne(
		ctx,
		idFilter(topic),
		options.FindOne().SetProjection(bson.D{{Key: "stale_token", Value: 1}}),
	).Decode(&doc)
	return doc.StaleToken != staleTokenInt, err
}

func (s *MongoDB) StorePushCert(ctx context.Context, pemCert, pemKey []byte) error {
	topic, err := cryptoutil.TopicFromPEMCert(pemCert)
	if err != nil {
		return err
	}

	t := now()
	res, err := s.pushCerts.UpdateOne(
		ctx,
		idFilter(topic),
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "cert_pem", Value: pemCert},
				{Key: "key_pem", Value: pemKey},
				{Key: "updated_at", Value: t},
			}},
			{Key: "$inc", Value: bson.D{{Key: "stale_token", Value: 1}}},
		},
	)
	if err != nil || res.MatchedCount > 0 {
		return err
	}

	_, err = s.pushCerts.InsertOne(ctx, pushCertDocument{
		ID:         topic,
		CertPEM:    pemCert,
		KeyPEM:     pemKey,
		StaleToken: 0,
		CreatedAt:  t,
		UpdatedAt:  t,
	})
	if err != nil && !isDuplicateKey(err) {
		return err
	}
	if isDuplicateKey(err) {
		_, err = s.pushCerts.UpdateOne(
			ctx,
			idFilter(topic),
			bson.D{
				{Key: "$set", Value: bson.D{
					{Key: "cert_pem", Value: pemCert},
					{Key: "key_pem", Value: pemKey},
					{Key: "updated_at", Value: t},
				}},
				{Key: "$inc", Value: bson.D{{Key: "stale_token", Value: 1}}},
			},
		)
	}
	return err
}
