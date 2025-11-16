package main

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/lib/pq"
	"github.com/sandrolain/eventkit/pkg/common"
	"github.com/sandrolain/eventkit/pkg/testpayload"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func sendCommand() *cobra.Command {
	var (
		connStr        string
		channel        string
		interval       string
		payload        string
		mime           string
		seed           int64
		allowFileReads bool
		templateVars   []string
		fileRoot       string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Periodically send NOTIFY to PostgreSQL channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			db, err := sql.Open("postgres", connStr)
			if err != nil {
				return fmt.Errorf("DB open error: %w", err)
			}
			defer func() {
				if err := db.Close(); err != nil {
					slog.Error("Failed to close DB connection", "error", err)
				}
			}()

			logger := toolutil.Logger()
			if seed != 0 {
				testpayload.SeedRandom(seed)
			}
			testpayload.SetAllowFileReads(allowFileReads)
			testpayload.SetFileRoot(fileRoot)
			varsMap, errVars := toolutil.ParseTemplateVars(templateVars)
			if errVars != nil {
				return fmt.Errorf("invalid template-var: %w", errVars)
			}
			testpayload.SetTemplateVars(varsMap)

			logger.Info("Sending NOTIFY to PostgreSQL", "channel", channel, "interval", interval)

			return common.StartPeriodicTask(ctx, interval, func() error {
				b, _, err := toolutil.BuildPayload(payload, mime)
				if err != nil {
					logger.Error("Failed to build payload", "error", err)
					return err
				}

				// NOTIFY doesn't support parameterized queries, so we must build the SQL string directly
				// Use pq.QuoteLiteral for safe escaping
				notifySQL := fmt.Sprintf("NOTIFY %s, %s", pq.QuoteIdentifier(channel), pq.QuoteLiteral(string(b)))
				if _, err := db.Exec(notifySQL); err != nil {
					logger.Error("NOTIFY error", "error", err)
					return err
				}

				logger.Info("NOTIFY sent", "channel", channel, "bytes", len(b))
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&connStr, "conn", "postgres://user:pass@localhost:5432/postgres?sslmode=disable", "PostgreSQL connection string")
	cmd.Flags().StringVar(&channel, "channel", "test_channel", "NOTIFY channel name")
	toolutil.AddPayloadFlags(cmd, &payload, "{nowtime}", &mime, toolutil.CTText)
	toolutil.AddIntervalFlag(cmd, &interval, "5s")
	toolutil.AddSeedFlag(cmd, &seed)
	toolutil.AddAllowFileReadsFlag(cmd, &allowFileReads)
	toolutil.AddTemplateVarFlag(cmd, &templateVars)
	toolutil.AddFileRootFlag(cmd, &fileRoot)

	return cmd
}
