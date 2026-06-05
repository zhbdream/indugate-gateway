package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/indugate/gateway/internal/model"
	"gorm.io/gorm"
)

var (
	devicesTotal     atomic.Int64
	devicesConnected atomic.Int64
	activeAlerts     atomic.Int64
	httpRequests     atomic.Int64
)

func Register() {}

func StartRefresh(ctx context.Context, db *gorm.DB, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	go func() {
		refresh(db)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				refresh(db)
			}
		}
	}()
}

func refresh(db *gorm.DB) {
	ctx := context.Background()
	var total, connected, alerts int64
	db.WithContext(ctx).Model(&model.Device{}).Count(&total)
	db.WithContext(ctx).Model(&model.Device{}).Where("status = ?", model.DeviceStatusConnected).Count(&connected)
	db.WithContext(ctx).Model(&model.AlertEvent{}).Where("status = ?", model.AlertStatusActive).Count(&alerts)
	devicesTotal.Store(total)
	devicesConnected.Store(connected)
	activeAlerts.Store(alerts)
}

func ObserveHTTPRequest() {
	httpRequests.Add(1)
}

func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprintf(w, "# HELP indugate_devices_total Total number of devices\n")
		fmt.Fprintf(w, "# TYPE indugate_devices_total gauge\n")
		fmt.Fprintf(w, "indugate_devices_total %d\n", devicesTotal.Load())
		fmt.Fprintf(w, "# HELP indugate_devices_connected Number of connected devices\n")
		fmt.Fprintf(w, "# TYPE indugate_devices_connected gauge\n")
		fmt.Fprintf(w, "indugate_devices_connected %d\n", devicesConnected.Load())
		fmt.Fprintf(w, "# HELP indugate_active_alerts Number of active alert events\n")
		fmt.Fprintf(w, "# TYPE indugate_active_alerts gauge\n")
		fmt.Fprintf(w, "indugate_active_alerts %d\n", activeAlerts.Load())
		fmt.Fprintf(w, "# HELP indugate_http_requests_total Total HTTP requests observed\n")
		fmt.Fprintf(w, "# TYPE indugate_http_requests_total counter\n")
		fmt.Fprintf(w, "indugate_http_requests_total %d\n", httpRequests.Load())
	})
}
