package cartstore

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpannerCartStore(t *testing.T) {
	// Set emulator host env var for the client
	os.Setenv("SPANNER_EMULATOR_HOST", "localhost:9010")

	ctx := context.Background()
	projectID := "test-project"
	instanceID := "test-instance"
	databaseID := "test-database"
	tableName := "CartItems"
	dbString := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)

	// Admin clients to create instance/db
	instanceAdmin, err := instance.NewInstanceAdminClient(ctx)
	require.NoError(t, err)
	defer instanceAdmin.Close()

	databaseAdmin, err := database.NewDatabaseAdminClient(ctx)
	require.NoError(t, err)
	defer databaseAdmin.Close()

	// Create Instance (ignore 'already exists' errors for local dev)
	_, err = instanceAdmin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     "projects/" + projectID,
		InstanceId: instanceID,
		Instance: &instancepb.Instance{
			Config:      "emulator-config",
			DisplayName: "Test Instance",
			NodeCount:   1,
		},
	})
	// In a real retry loop or check 'AlreadyExists'
	// For this test script we assume clean slate or ignore

	// Create Database with Schema
	op, err := databaseAdmin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", projectID, instanceID),
		CreateStatement: "CREATE DATABASE `" + databaseID + "`",
		ExtraStatements: []string{
			fmt.Sprintf(`CREATE TABLE %s (
				userId STRING(1024),
				productId STRING(1024),
				quantity INT64,
			) PRIMARY KEY (userId, productId)`, tableName),
		},
	})
	if err == nil {
		_, err = op.Wait(ctx)
		require.NoError(t, err)
	}

	// Give emulator a moment
	time.Sleep(1 * time.Second)

	// Create Store
	store, err := NewSpannerCartStore(ctx, dbString, tableName)
	require.NoError(t, err)
	defer store.Close()

	userID := "test-user-spanner"

	// Clean table first
	_, err = store.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{spanner.Delete(tableName, spanner.AllKeys())})
	})
	require.NoError(t, err)

	t.Run("AddItem", func(t *testing.T) {
		err := store.AddItem(ctx, userID, "item-1", 1)
		require.NoError(t, err)

		cart, err := store.GetCart(ctx, userID)
		require.NoError(t, err)
		require.Len(t, cart.Items, 1)
		assert.Equal(t, "item-1", cart.Items[0].ProductId)
		assert.Equal(t, int32(1), cart.Items[0].Quantity)
	})

	t.Run("AddItem increment", func(t *testing.T) {
		err := store.AddItem(ctx, userID, "item-1", 2)
		require.NoError(t, err)

		cart, err := store.GetCart(ctx, userID)
		require.NoError(t, err)
		require.Len(t, cart.Items, 1)
		assert.Equal(t, int32(3), cart.Items[0].Quantity)
	})

	t.Run("EmptyCart", func(t *testing.T) {
		err := store.EmptyCart(ctx, userID)
		require.NoError(t, err)

		cart, err := store.GetCart(ctx, userID)
		require.NoError(t, err)
		assert.Nil(t, cart.Items)
	})
}
