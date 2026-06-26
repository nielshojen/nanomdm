// Package mongodb implements a MongoDB-backed NanoMDM storage backend.
package mongodb

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const defaultDatabase = "nanomdm"

var ErrMissingDSN = errors.New("missing MongoDB DSN")

// MongoDB is a MongoDB-backed storage backend.
type MongoDB struct {
	client          *mongo.Client
	db              *mongo.Database
	rm              bool
	devices         *mongo.Collection
	users           *mongo.Collection
	enrollments     *mongo.Collection
	commands        *mongo.Collection
	commandResults  *mongo.Collection
	enrollmentQueue *mongo.Collection
	pushCerts       *mongo.Collection
	certAuth        *mongo.Collection
}

type config struct {
	dsn              string
	client           *mongo.Client
	database         string
	collectionPrefix string
	rm               bool
}

type collectionNames struct {
	devices         string
	users           string
	enrollments     string
	commands        string
	commandResults  string
	enrollmentQueue string
	pushCerts       string
	certAuth        string
}

// Option configures a MongoDB storage backend.
type Option func(*config)

// WithDSN configures the MongoDB connection string.
func WithDSN(dsn string) Option {
	return func(c *config) {
		c.dsn = dsn
	}
}

// WithClient configures an existing MongoDB client.
func WithClient(client *mongo.Client) Option {
	return func(c *config) {
		c.client = client
	}
}

// WithDatabase configures the MongoDB database name.
func WithDatabase(database string) Option {
	return func(c *config) {
		c.database = database
	}
}

// WithCollectionPrefix configures a prefix for NanoMDM collection names.
func WithCollectionPrefix(prefix string) Option {
	return func(c *config) {
		c.collectionPrefix = prefix
	}
}

// WithDeleteCommands deletes commands and command responses after a non-NotNow response.
func WithDeleteCommands() Option {
	return func(c *config) {
		c.rm = true
	}
}

// New creates a new MongoDB-backed NanoMDM storage backend.
func New(opts ...Option) (*MongoDB, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	var err error
	createdClient := cfg.client == nil
	if cfg.client == nil {
		if cfg.dsn == "" {
			return nil, ErrMissingDSN
		}
		cfg.client, err = mongo.Connect(
			options.Client().ApplyURI(cfg.dsn).SetAppName("nanomdm"),
		)
		if err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = cfg.client.Ping(ctx, nil); err != nil {
		if createdClient {
			_ = cfg.client.Disconnect(context.Background())
		}
		return nil, err
	}

	database := cfg.database
	if database == "" {
		database = databaseFromDSN(cfg.dsn)
	}
	if database == "" {
		database = defaultDatabase
	}

	db := cfg.client.Database(database)
	names := newCollectionNames(cfg.collectionPrefix)
	s := &MongoDB{
		client:          cfg.client,
		db:              db,
		rm:              cfg.rm,
		devices:         db.Collection(names.devices),
		users:           db.Collection(names.users),
		enrollments:     db.Collection(names.enrollments),
		commands:        db.Collection(names.commands),
		commandResults:  db.Collection(names.commandResults),
		enrollmentQueue: db.Collection(names.enrollmentQueue),
		pushCerts:       db.Collection(names.pushCerts),
		certAuth:        db.Collection(names.certAuth),
	}
	if err = s.ensureIndexes(ctx); err != nil {
		if createdClient {
			_ = cfg.client.Disconnect(context.Background())
		}
		return nil, err
	}
	return s, nil
}

// Client returns the MongoDB client used by the storage backend.
func (s *MongoDB) Client() *mongo.Client {
	return s.client
}

// Database returns the MongoDB database used by the storage backend.
func (s *MongoDB) Database() *mongo.Database {
	return s.db
}

func databaseFromDSN(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(u.Path, "/")
}

func newCollectionNames(prefix string) collectionNames {
	return collectionNames{
		devices:         prefix + "devices",
		users:           prefix + "users",
		enrollments:     prefix + "enrollments",
		commands:        prefix + "commands",
		commandResults:  prefix + "command_results",
		enrollmentQueue: prefix + "enrollment_queue",
		pushCerts:       prefix + "push_certs",
		certAuth:        prefix + "cert_auth_associations",
	}
}

func (s *MongoDB) collections() []*mongo.Collection {
	return []*mongo.Collection{
		s.devices,
		s.users,
		s.enrollments,
		s.commands,
		s.commandResults,
		s.enrollmentQueue,
		s.pushCerts,
		s.certAuth,
	}
}

func idFilter(id string) bson.D {
	return bson.D{{Key: "_id", Value: id}}
}

func compositeKey(parts ...string) string {
	return strings.Join(parts, "\x00")
}

func now() time.Time {
	return time.Now().UTC()
}

func isDuplicateKey(err error) bool {
	return mongo.IsDuplicateKeyError(err)
}
