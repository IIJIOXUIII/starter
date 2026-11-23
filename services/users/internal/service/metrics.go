// Package service — метрики users-service (Prometheus).
package service

import "github.com/prometheus/client_golang/prometheus"

// profilesCreatedTotal — созданные профили по источнику (event | grpc | http | upsert).
var profilesCreatedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "users_profiles_created_total",
	Help: "Profiles created by source (event/grpc/http/upsert).",
}, []string{"source"})

func init() {
	prometheus.MustRegister(profilesCreatedTotal)
}

// IncProfilesCreated увеличивает счётчик создания профиля.
// Экспортируется транспортным слоям (gRPC/HTTP).
func IncProfilesCreated(source string) {
	profilesCreatedTotal.WithLabelValues(source).Inc()
}
