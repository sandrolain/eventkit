package main

import (
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sandrolain/eventkit/pkg/common"
	"github.com/sandrolain/eventkit/pkg/testpayload"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

const (
	tcpPrefix = "tcp://"
	sslPrefix = "ssl://"
	wsPrefix  = "ws://"
)

func sendCommand() *cobra.Command {
	var (
		sendBroker     string
		sendTopic      string
		sendPayload    string
		sendMIME       string
		sendInterval   string
		sendQoS        int
		sendRetain     bool
		sendClientID   string
		headers        []string
		openDelim      string
		closeDelim     string
		seed           int64
		allowFileReads bool
		templateVars   []string
		fileRoot       string
		cacheFiles     bool
		once           bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Publish periodic MQTT messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			if !strings.HasPrefix(sendBroker, tcpPrefix) && !strings.HasPrefix(sendBroker, sslPrefix) && !strings.HasPrefix(sendBroker, wsPrefix) {
				sendBroker = tcpPrefix + sendBroker
			}
			opts := mqtt.NewClientOptions().AddBroker(sendBroker)
			if sendClientID == "" {
				sendClientID = fmt.Sprintf("mqttcli-pub-%d", time.Now().UnixNano())
			}
			opts.SetClientID(sendClientID).SetAutoReconnect(true)
			client := mqtt.NewClient(opts)
			if token := client.Connect(); token.Wait() && token.Error() != nil {
				return fmt.Errorf("MQTT connection error: %w", token.Error())
			}
			defer client.Disconnect(250)

			toolutil.PrintSuccess("Connected to MQTT broker")
			toolutil.PrintKeyValue("Broker", sendBroker)
			toolutil.PrintKeyValue("Topic", sendTopic)
			toolutil.PrintKeyValue("QoS", sendQoS)
			toolutil.PrintKeyValue("Interval", sendInterval)

			if seed != 0 {
				testpayload.SeedRandom(seed)
			}
			testpayload.SetAllowFileReads(allowFileReads)
			testpayload.SetFileRoot(fileRoot)
			testpayload.SetFileCacheEnabled(cacheFiles)
			varsMap, errVars := toolutil.ParseTemplateVars(templateVars)
			if errVars != nil {
				return fmt.Errorf("invalid template-var: %w", errVars)
			}
			testpayload.SetTemplateVars(varsMap)

			_, errHeaders := toolutil.ParseHeadersWithDelimiters(headers, openDelim, closeDelim)
			if errHeaders != nil {
				return fmt.Errorf("invalid headers: %w", errHeaders)
			}
			// Note: MQTT v5 user properties can be set from headers

			publish := func() error {
				body, _, err := toolutil.BuildPayloadWithDelimiters(sendPayload, sendMIME, openDelim, closeDelim)
				if err != nil {
					toolutil.PrintError("Payload build error: %v", err)
					return err
				}
				token := client.Publish(sendTopic, byte(sendQoS), sendRetain, body)
				token.Wait()
				if token.Error() != nil {
					toolutil.PrintError("Publish error: %v", token.Error())
					return token.Error()
				}
				toolutil.PrintInfo("Published %d bytes to %s", len(body), sendTopic)
				return nil
			}

			return common.RunOnceOrPeriodic(ctx, once, sendInterval, publish)
		},
	}

	cmd.Flags().StringVar(&sendBroker, "broker", "tcp://localhost:1883", "MQTT broker URL (tcp://host:port)")
	cmd.Flags().StringVar(&sendTopic, "topic", "test/topic", "MQTT topic to publish to")
	cmd.Flags().IntVar(&sendQoS, "qos", 0, "MQTT QoS level (0,1,2)")
	cmd.Flags().BoolVar(&sendRetain, "retain", false, "Retain messages")
	cmd.Flags().StringVar(&sendClientID, "clientid", "", "Client ID (auto if empty)")
	toolutil.AddPayloadFlags(cmd, &sendPayload, "{}", &sendMIME, toolutil.CTText)
	toolutil.AddIntervalFlag(cmd, &sendInterval, "5s")
	toolutil.AddOnceFlag(cmd, &once)
	toolutil.AddHeadersFlag(cmd, &headers)
	toolutil.AddTemplateDelimiterFlags(cmd, &openDelim, &closeDelim)
	toolutil.AddSeedFlag(cmd, &seed)
	toolutil.AddAllowFileReadsFlag(cmd, &allowFileReads)
	toolutil.AddTemplateVarFlag(cmd, &templateVars)
	toolutil.AddFileRootFlag(cmd, &fileRoot)
	toolutil.AddFileCacheFlag(cmd, &cacheFiles)

	return cmd
}
