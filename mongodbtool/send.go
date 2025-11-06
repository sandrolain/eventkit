package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func sendCommand() *cobra.Command {
	var (
		uri        string
		database   string
		collection string
		payload    string
		mime       string
		interval   string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Insert documents into MongoDB periodically",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			// Connect to MongoDB
			clientOpts := options.Client().ApplyURI(uri)
			client, err := mongo.Connect(ctx, clientOpts)
			if err != nil {
				return fmt.Errorf("failed to connect to MongoDB: %w", err)
			}
			defer func() {
				if err := client.Disconnect(context.Background()); err != nil {
					toolutil.PrintError("Failed to disconnect: %v", err)
				}
			}()

			// Ping to verify connection
			if err := client.Ping(ctx, nil); err != nil {
				return fmt.Errorf("failed to ping MongoDB: %w", err)
			}

			coll := client.Database(database).Collection(collection)

			toolutil.PrintSuccess("Connected to MongoDB")
			toolutil.PrintKeyValue("URI", uri)
			toolutil.PrintKeyValue("Database", database)
			toolutil.PrintKeyValue("Collection", collection)
			toolutil.PrintKeyValue("Interval", interval)

			insert := func() error {
				body, _, err := toolutil.BuildPayload(payload, mime)
				if err != nil {
					toolutil.PrintError("Payload build error: %v", err)
					return err
				}

				// Parse JSON to BSON document
				var doc bson.M
				if err := bson.UnmarshalExtJSON(body, true, &doc); err != nil {
					toolutil.PrintError("Failed to parse JSON: %v", err)
					return err
				}

				// Add timestamp
				doc["_insertedAt"] = time.Now()

				insertCtx, insertCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer insertCancel()

				result, err := coll.InsertOne(insertCtx, doc)
				if err != nil {
					toolutil.PrintError("Insert error: %v", err)
					return err
				}

				toolutil.PrintInfo("Inserted document with ID: %v", result.InsertedID)
				return nil
			}

			return common.StartPeriodicTask(ctx, interval, insert)
		},
	}

	cmd.Flags().StringVar(&uri, "uri", "mongodb://localhost:27017", "MongoDB connection URI")
	cmd.Flags().StringVar(&database, "database", "test", "Database name")
	cmd.Flags().StringVar(&collection, "collection", "events", "Collection name")
	toolutil.AddPayloadFlags(cmd, &payload, `{"message":"{sentence}","timestamp":"{nowtime}"}`, &mime, toolutil.CTJSON)
	toolutil.AddIntervalFlag(cmd, &interval, "5s")

	return cmd
}
