package service

import (
	"context"
	"fmt"

	"github.com/indugate/gateway/internal/config"
	"github.com/indugate/gateway/internal/protocol"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	influxapi "github.com/influxdata/influxdb-client-go/v2/api"
	"go.uber.org/zap"
)

type InfluxWriter struct {
	log    *zap.Logger
	cfg    config.InfluxDBConfig
	client influxdb2.Client
	write  influxapi.WriteAPI
}

func NewInfluxWriter(log *zap.Logger, cfg config.InfluxDBConfig) *InfluxWriter {
	if !cfg.Enabled || cfg.URL == "" || cfg.Token == "" {
		return nil
	}
	client := influxdb2.NewClient(cfg.URL, cfg.Token)
	w := &InfluxWriter{
		log:    log,
		cfg:    cfg,
		client: client,
		write:  client.WriteAPI(cfg.Org, cfg.Bucket),
	}
	log.Info("influxdb writer enabled", zap.String("url", cfg.URL), zap.String("bucket", cfg.Bucket))
	return w
}

func (w *InfluxWriter) WriteTelemetry(deviceID uint, nodeID string, value *protocol.DataValue) {
	if w == nil || value == nil {
		return
	}
	p := influxdb2.NewPointWithMeasurement("telemetry").
		AddTag("device_id", fmt.Sprintf("%d", deviceID)).
		AddTag("node_id", nodeID).
		AddField("value_raw", fmt.Sprintf("%v", value.Value)).
		SetTime(value.Timestamp)
	if num, ok := toFloat64(value.Value); ok {
		p.AddField("value", num)
	}
	w.write.WritePoint(p)
}

func (w *InfluxWriter) Close() {
	if w == nil {
		return
	}
	w.write.Flush()
	w.client.Close()
}

type HistoryRecorder struct {
	history *HistoryService
	alerts  *AlertService
	influx  *InfluxWriter
}

func NewHistoryRecorder(history *HistoryService, alerts *AlertService, influx *InfluxWriter) *HistoryRecorder {
	return &HistoryRecorder{history: history, alerts: alerts, influx: influx}
}

func (r *HistoryRecorder) RecordRead(ctx context.Context, deviceID uint, value *protocol.DataValue) {
	if value == nil {
		return
	}
	if r.history != nil {
		_ = r.history.Record(ctx, deviceID, value)
	}
	if r.influx != nil {
		r.influx.WriteTelemetry(deviceID, value.NodeID, value)
	}
	if r.alerts != nil {
		_, _ = r.alerts.Evaluate(ctx, deviceID, value.NodeID, value.Value)
	}
}

func (r *HistoryRecorder) RecordEvent(ctx context.Context, deviceID uint, evt protocol.DataChangeEvent) {
	dv := &protocol.DataValue{
		NodeID:    evt.NodeID,
		Value:     evt.Value,
		Status:    evt.Status,
		Timestamp: evt.Timestamp,
	}
	r.RecordRead(ctx, deviceID, dv)
}
