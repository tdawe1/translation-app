package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gengowatcher_jobs_created_total",
		Help: "Total number of translation jobs created",
	})

	JobsCompleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gengowatcher_jobs_completed_total",
		Help: "Total number of translation jobs completed",
	})

	ActiveUsers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gengowatcher_active_users",
		Help: "Number of currently active users",
	})

	TranslationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "gengowatcher_translation_duration_seconds",
		Help:    "Time taken to process translation jobs",
		Buckets: []float64{1, 5, 10, 30, 60, 120, 300},
	})
)
