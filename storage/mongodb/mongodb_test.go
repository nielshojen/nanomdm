package mongodb

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/micromdm/nanomdm/test/e2e"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestDatabaseFromDSN(t *testing.T) {
	for _, tt := range []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "database path",
			dsn:  "mongodb://localhost:27017/nanomdm",
			want: "nanomdm",
		},
		{
			name: "database path with auth source",
			dsn:  "mongodb://user:pass@localhost:27017/nanomdm?authSource=admin",
			want: "nanomdm",
		},
		{
			name: "escaped database path",
			dsn:  "mongodb://localhost:27017/nano%2Dmdm",
			want: "nano-mdm",
		},
		{
			name: "no database",
			dsn:  "mongodb://localhost:27017",
			want: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := databaseFromDSN(tt.dsn); got != tt.want {
				t.Fatalf("databaseFromDSN(%q) = %q, want %q", tt.dsn, got, tt.want)
			}
		})
	}
}

func TestMongoDB(t *testing.T) {
	testDSN := os.Getenv("NANOMDM_MONGODB_STORAGE_TEST_DSN")
	if testDSN == "" {
		t.Skip("NANOMDM_MONGODB_STORAGE_TEST_DSN not set")
	}

	ctx := context.Background()
	prefix := "test_" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_"

	s, err := New(WithDSN(testDSN), WithCollectionPrefix(prefix))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		names := newCollectionNames(prefix)
		for _, name := range []string{
			names.devices,
			names.users,
			names.enrollments,
			names.commands,
			names.commandResults,
			names.enrollmentQueue,
			names.pushCerts,
			names.certAuth,
		} {
			if err := s.Database().Collection(name).Drop(ctx); err != nil {
				t.Logf("dropping collection %s: %v", name, err)
			}
		}
	})

	requireIndex(t, ctx, s.devices, "serial_number")
	requireIndex(t, ctx, s.enrollments, "type")
	requireIndex(t, ctx, s.commandResults, "status")
	requireIndex(t, ctx, s.enrollmentQueue, "id_active_priority_created_at")
	requireIndex(t, ctx, s.certAuth, "sha256")

	t.Run("e2e", func(t *testing.T) { e2e.TestE2E(t, ctx, s) })
}

func requireIndex(t *testing.T, ctx context.Context, coll *mongo.Collection, name string) {
	t.Helper()
	specs, err := coll.Indexes().ListSpecifications(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range specs {
		if spec.Name == name {
			return
		}
	}
	t.Fatalf("index %q not found on collection %q", name, coll.Name())
}
