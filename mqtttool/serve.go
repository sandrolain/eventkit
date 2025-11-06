package main

import (
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func serveCommand() *cobra.Command {
	var (
		subBroker   string
		subTopic    string
		subClientID string
		subQoS      int
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Subscribe to a topic and log messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !strings.HasPrefix(subBroker, tcpPrefix) && !strings.HasPrefix(subBroker, sslPrefix) && !strings.HasPrefix(subBroker, wsPrefix) {
				subBroker = tcpPrefix + subBroker
			}
			if subClientID == "" {
				subClientID = fmt.Sprintf("mqttcli-sub-%d", time.Now().UnixNano())
			}

			opts := mqtt.NewClientOptions().AddBroker(subBroker).SetClientID(subClientID)
			client := mqtt.NewClient(opts)
			if token := client.Connect(); token.Wait() && token.Error() != nil {
				return fmt.Errorf("error connecting to MQTT broker: %w", token.Error())
			}
			defer client.Disconnect(250)

			toolutil.PrintSuccess("Subscribed to MQTT topic")
			toolutil.PrintKeyValue("Broker", subBroker)
			toolutil.PrintKeyValue("Topic", subTopic)
			toolutil.PrintKeyValue("QoS", subQoS)

			if token := client.Subscribe(subTopic, byte(subQoS), func(_ mqtt.Client, msg mqtt.Message) {
				ct := toolutil.GuessMIME(msg.Payload())
				sections := []toolutil.MessageSection{
					{Title: "Topic", Items: []toolutil.KV{{Key: "Name", Value: msg.Topic()}}},
				}
				toolutil.PrintColoredMessage("MQTT", sections, msg.Payload(), ct)
			}); token.Wait() && token.Error() != nil {
				return fmt.Errorf("error subscribing to topic: %w", token.Error())
			}

			common.WaitForShutdown()
			return nil
		},
	}

	cmd.Flags().StringVar(&subBroker, "broker", "tcp://localhost:1883", "MQTT broker URL (tcp://host:port)")
	cmd.Flags().StringVar(&subTopic, "topic", "test/topic", "MQTT topic to subscribe to")
	cmd.Flags().StringVar(&subClientID, "clientid", "", "Client ID (auto if empty)")
	cmd.Flags().IntVar(&subQoS, "qos", 0, "MQTT QoS level (0,1,2)")

	return cmd
}
