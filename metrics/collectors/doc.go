// Package collectors provides generic collector builders that automatically
// collect metrics from any datasource implementing the CustomMetricSource interface.
//
// # Overview
//
// The collectors package simplifies metric collection by providing reusable
// builders that wrap the core metrics system. Instead of manually creating
// and updating metrics, implement the CustomMetricSource interface and let
// the builder handle everything automatically.
//
// # Features
//
//   - Automatic metric creation (lazy initialization)
//   - Counter delta tracking (handles resets gracefully)
//   - Both pull (polling) and push (event-driven) collection
//   - Thread-safe concurrent access
//   - Configurable collection intervals
//   - Support for all metric types (Counter, Gauge, Histogram, Summary, Timer)
//
// # Basic Usage
//
// Implement CustomMetricSource for your datasource:
//
//	type DBPoolSource struct {
//	    db *sql.DB
//	}
//
//	func (s *DBPoolSource) Name() string {
//	    return "database_pool"
//	}
//
//	func (s *DBPoolSource) Collect(ctx context.Context) (*MetricSnapshot, error) {
//	    stats := s.db.Stats()
//	    return &MetricSnapshot{
//	        Gauges: map[string]float64{
//	            "connections_open":   float64(stats.OpenConnections),
//	            "connections_in_use": float64(stats.InUse),
//	            "connections_idle":   float64(stats.Idle),
//	        },
//	        Counters: map[string]float64{
//	            "wait_count":     float64(stats.WaitCount),
//	            "wait_duration":  float64(stats.WaitDuration),
//	        },
//	    }, nil
//	}
//
// Create and start the collector:
//
//	source := &DBPoolSource{db: db}
//	collector := collectors.NewCustomCollectorBuilder(source).
//	    WithInterval(5 * time.Second).
//	    WithOptions(
//	        metrics.WithNamespace("db"),
//	        metrics.WithSubsystem("postgres"),
//	    )
//
//	collector.Start()
//	defer collector.Stop()
//
//	// Access the underlying collector
//	metricsCollector := collector.Metrics()
//
// # Pull vs Push
//
// Pull (Polling): The builder periodically calls Collect() on your datasource.
// Use CustomCollectorBuilder for pull-based collection.
//
// Push (Event-driven): Your datasource pushes snapshots when events occur.
// Use PushableCollectorBuilder for push-based collection.
//
// Hybrid: PushableCollectorBuilder supports both - periodic polling plus
// on-demand pushes for important events.
//
// # Push Example
//
//	collector := collectors.NewPushableCollectorBuilder(source).
//	    WithInterval(10 * time.Second).  // Background polling
//	    WithBufferSize(500)               // Push buffer size
//
//	collector.Start()
//	defer collector.Stop()
//
//	// Push metrics on-demand
//	snapshot := &collectors.MetricSnapshot{
//	    Counters: map[string]float64{
//	        "critical_event": 1,
//	    },
//	}
//	collector.Push(snapshot)
//
// # Counter Delta Tracking
//
// The builder automatically tracks counter deltas. If your datasource returns
// monotonically increasing counters (like cumulative totals), the builder
// calculates the delta between collections:
//
//	Collection 1: counter = 100  -> Records: 100
//	Collection 2: counter = 150  -> Records: 50 (delta)
//	Collection 3: counter = 200  -> Records: 50 (delta)
//
// If a counter reset is detected (new value < old value), the builder treats
// the new value as the delta to avoid negative values.
//
// # Lazy Metric Creation
//
// Metrics are created on-demand during collection. Your datasource can
// dynamically add or remove metrics between collections:
//
//	First collect:  Counters: {"metric_a": 10}
//	Second collect: Counters: {"metric_a": 20, "metric_b": 5}
//	                // metric_b is automatically created
//
// # Thread Safety
//
// All operations are thread-safe. Multiple goroutines can:
//   - Call CollectOnce() concurrently
//   - Push() metrics simultaneously (PushableCollectorBuilder)
//   - Access the underlying Metrics() concurrently
//
// # Error Handling
//
// Collection errors from your datasource are logged but don't stop the
// collection loop. This allows transient errors without losing all metrics.
//
// # Best Practices
//
//  1. Keep Collect() fast - it's called on every interval
//  2. Use gauges for absolute values (memory, connections, queue depth)
//  3. Use counters for cumulative totals (requests, errors, bytes)
//  4. Return nil maps for metric types you don't use
//  5. Set appropriate collection intervals (default 10s)
//  6. Use push for critical events that need immediate collection
//  7. Configure buffer size based on push rate (default 100)
//
// # Redis Example
//
//	type RedisSource struct {
//	    client redis.UniversalClient
//	}
//
//	func (r *RedisSource) Name() string {
//	    return "redis"
//	}
//
//	func (r *RedisSource) Collect(ctx context.Context) (*MetricSnapshot, error) {
//	    info := r.client.Info(ctx, "stats", "memory")
//	    stats := parseRedisInfo(info.Val())
//
//	    return &MetricSnapshot{
//	        Gauges: map[string]float64{
//	            "memory_used_bytes":  stats["used_memory"],
//	            "connected_clients":  stats["connected_clients"],
//	            "keys_total":         stats["keys"],
//	        },
//	        Counters: map[string]float64{
//	            "commands_processed": stats["total_commands_processed"],
//	            "keyspace_hits":      stats["keyspace_hits"],
//	            "keyspace_misses":    stats["keyspace_misses"],
//	        },
//	    }, nil
//	}
//
// # Custom Application Metrics
//
//	type AppMetricsSource struct {
//	    app *MyApplication
//	}
//
//	func (a *AppMetricsSource) Name() string {
//	    return "myapp"
//	}
//
//	func (a *AppMetricsSource) Collect(ctx context.Context) (*MetricSnapshot, error) {
//	    return &MetricSnapshot{
//	        Gauges: map[string]float64{
//	            "active_sessions": float64(a.app.ActiveSessions()),
//	            "queue_depth":     float64(a.app.QueueDepth()),
//	        },
//	        Counters: map[string]float64{
//	            "requests_total": float64(a.app.TotalRequests()),
//	            "errors_total":   float64(a.app.TotalErrors()),
//	        },
//	        Histograms: map[string][]float64{
//	            "response_times": a.app.GetRecentResponseTimes(),
//	        },
//	    }, nil
//	}
package collectors
