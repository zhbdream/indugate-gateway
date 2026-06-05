package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/model"
	"go.uber.org/zap"
)

type AlertNotifier struct {
	log  *zap.Logger
	cfg  config.AlertConfig
	mu   sync.Mutex
	mqtt mqtt.Client
}

func NewAlertNotifier(log *zap.Logger, cfg config.AlertConfig) *AlertNotifier {
	if cfg.WebhookURL == "" && !cfg.MQTTEnabled {
		return nil
	}
	return &AlertNotifier{log: log, cfg: cfg}
}

func (n *AlertNotifier) Notify(ctx context.Context, event model.AlertEvent) {
	if n == nil {
		return
	}
	go n.dispatch(event)
}

func (n *AlertNotifier) dispatch(event model.AlertEvent) {
	payload, err := json.Marshal(map[string]any{
		"id":           event.ID,
		"rule_id":      event.RuleID,
		"device_id":    event.DeviceID,
		"node_id":      event.NodeID,
		"level":        event.Level,
		"message":      event.Message,
		"value":        event.Value,
		"status":       event.Status,
		"triggered_at": event.TriggeredAt,
	})
	if err != nil {
		n.log.Warn("marshal alert payload", zap.Error(err))
		return
	}

	if n.cfg.WebhookURL != "" {
		n.postWebhook(payload)
	}
	if n.cfg.MQTTEnabled {
		n.publishMQTT(payload)
	}
}

func (n *AlertNotifier) postWebhook(payload []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.cfg.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		n.log.Warn("create webhook request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		n.log.Warn("alert webhook failed", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		n.log.Warn("alert webhook non-success", zap.Int("status", resp.StatusCode))
	}
}

func (n *AlertNotifier) publishMQTT(payload []byte) {
	client, err := n.mqttClient()
	if err != nil {
		n.log.Warn("alert mqtt connect failed", zap.Error(err))
		return
	}
	topic := n.cfg.MQTTTopic
	if topic == "" {
		topic = "indugate/alerts"
	}
	token := client.Publish(topic, 1, false, payload)
	if token.Wait() && token.Error() != nil {
		n.log.Warn("alert mqtt publish failed", zap.Error(token.Error()))
	}
}

func (n *AlertNotifier) mqttClient() (mqtt.Client, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.mqtt != nil && n.mqtt.IsConnected() {
		return n.mqtt, nil
	}

	broker := n.cfg.MQTTBroker
	if broker == "" {
		broker = "tcp://localhost:1883"
	}
	clientID := n.cfg.MQTTClientID
	if clientID == "" {
		clientID = "indugate-alert-notifier"
	}

	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetConnectTimeout(5 * time.Second).
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("connect mqtt broker: %w", token.Error())
	}
	n.mqtt = client
	return client, nil
}

func (n *AlertNotifier) Close() {
	if n == nil {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.mqtt != nil && n.mqtt.IsConnected() {
		n.mqtt.Disconnect(250)
	}
}
