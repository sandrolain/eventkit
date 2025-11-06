package main

import (
	"context"
	"fmt"

	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func serveCommand() *cobra.Command {
	var (
		uri        string
		database   string
		collection string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Watch MongoDB collection for changes",
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

			toolutil.PrintSuccess("Watching MongoDB collection for changes")
			toolutil.PrintKeyValue("URI", uri)
			toolutil.PrintKeyValue("Database", database)
			toolutil.PrintKeyValue("Collection", collection)

			// Create change stream
			pipeline := mongo.Pipeline{}
			opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
			changeStream, err := coll.Watch(ctx, pipeline, opts)
			if err != nil {
				return fmt.Errorf("failed to create change stream: %w", err)
			}
			defer func() {
				if err := changeStream.Close(context.Background()); err != nil {
					toolutil.PrintError("Failed to close change stream: %v", err)
				}
			}()

			// Watch for changes
			for changeStream.Next(ctx) {
				var changeDoc bson.M
				if err := changeStream.Decode(&changeDoc); err != nil {
					toolutil.PrintError("Failed to decode change: %v", err)
					continue
				}

				// Extract operation type and document
				operationType := "unknown"
				if op, ok := changeDoc["operationType"].(string); ok {
					operationType = op
				}

				dbName := ""
				collName := ""
				if ns, ok := changeDoc["ns"].(bson.M); ok {
					if db, ok := ns["db"].(string); ok {
						dbName = db
					}
					if coll, ok := ns["coll"].(string); ok {
						collName = coll
					}
				}

				sections := []toolutil.MessageSection{
					{
						Title: "Change Event",
						Items: []toolutil.KV{
							{Key: "Operation", Value: operationType},
							{Key: "Database", Value: dbName},
							{Key: "Collection", Value: collName},
						},
					},
				}

				// Get document data
				var docData []byte
				if fullDoc, ok := changeDoc["fullDocument"].(bson.M); ok {
					if data, err := bson.MarshalExtJSON(fullDoc, true, false); err == nil {
						docData = data
					}
				} else if docKey, ok := changeDoc["documentKey"].(bson.M); ok {
					if data, err := bson.MarshalExtJSON(docKey, true, false); err == nil {
						docData = data
					}
				}

				toolutil.PrintColoredMessage("MongoDB", sections, docData, toolutil.CTJSON)
			}

			if err := changeStream.Err(); err != nil {
				return fmt.Errorf("change stream error: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&uri, "uri", "mongodb://localhost:27017", "MongoDB connection URI")
	cmd.Flags().StringVar(&database, "database", "test", "Database name")
	cmd.Flags().StringVar(&collection, "collection", "events", "Collection name")

	return cmd
}
