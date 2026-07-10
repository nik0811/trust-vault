package pkg

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trustvault_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	HTTPActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trustvault_http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// Database metrics
	DBConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trustvault_db_connections_open",
			Help: "Number of open database connections",
		},
	)

	DBConnectionsInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trustvault_db_connections_in_use",
			Help: "Number of database connections in use",
		},
	)

	DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trustvault_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trustvault_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation"},
	)

	// Kafka metrics
	KafkaMessagesProduced = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_kafka_messages_produced_total",
			Help: "Total number of Kafka messages produced",
		},
		[]string{"topic"},
	)

	KafkaMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_kafka_messages_consumed_total",
			Help: "Total number of Kafka messages consumed",
		},
		[]string{"topic", "consumer_group"},
	)

	KafkaConsumerLag = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trustvault_kafka_consumer_lag",
			Help: "Kafka consumer lag (messages behind)",
		},
		[]string{"topic", "consumer_group", "partition"},
	)

	KafkaProducerErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_kafka_producer_errors_total",
			Help: "Total number of Kafka producer errors",
		},
		[]string{"topic"},
	)

	// Classification metrics
	ClassificationJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_classification_jobs_total",
			Help: "Total number of classification jobs processed",
		},
		[]string{"status", "model"},
	)

	ClassificationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "trustvault_classification_duration_seconds",
			Help:    "Classification job duration in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"model"},
	)

	ClassificationEntitiesFound = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_classification_entities_found_total",
			Help: "Total number of entities found during classification",
		},
		[]string{"entity_type"},
	)

	// Error metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type", "component"},
	)

	// AI Gate metrics
	GateQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_gate_queries_total",
			Help: "Total number of AI Gate queries",
		},
		[]string{"decision"},
	)

	GateQueryDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "trustvault_gate_query_duration_seconds",
			Help:    "AI Gate query duration in seconds",
			Buckets: []float64{.1, .25, .5, 1, 2.5, 5, 10, 30},
		},
	)

	// Audit metrics
	AuditLogsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trustvault_audit_logs_total",
			Help: "Total number of audit log entries",
		},
		[]string{"action", "resource"},
	)

	// Business metrics
	TenantsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trustvault_tenants_active",
			Help: "Number of active tenants",
		},
	)

	DataSourcesTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trustvault_datasources_total",
			Help: "Total number of data sources by status",
		},
		[]string{"status"},
	)

	PoliciesActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "trustvault_policies_active",
			Help: "Number of active governance policies",
		},
	)
)

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint string, status int, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, strconv.Itoa(status)).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordDBQuery records database query metrics
func RecordDBQuery(operation string, duration time.Duration) {
	DBQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordKafkaMessage records Kafka message metrics
func RecordKafkaMessage(topic string, produced bool) {
	if produced {
		KafkaMessagesProduced.WithLabelValues(topic).Inc()
	}
}

// RecordClassificationJob records classification job metrics
func RecordClassificationJob(status, model string, duration time.Duration, entitiesFound map[string]int) {
	ClassificationJobsTotal.WithLabelValues(status, model).Inc()
	ClassificationDuration.WithLabelValues(model).Observe(duration.Seconds())
	for entityType, count := range entitiesFound {
		ClassificationEntitiesFound.WithLabelValues(entityType).Add(float64(count))
	}
}

// RecordError records error metrics
func RecordError(errType, component string) {
	ErrorsTotal.WithLabelValues(errType, component).Inc()
}

// RecordGateQuery records AI Gate query metrics
func RecordGateQuery(decision string, duration time.Duration) {
	GateQueriesTotal.WithLabelValues(decision).Inc()
	GateQueryDuration.Observe(duration.Seconds())
}

// RecordAuditLog records audit log metrics
func RecordAuditLog(action, resource string) {
	AuditLogsTotal.WithLabelValues(action, resource).Inc()
}
