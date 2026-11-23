package http

// Метрики auth-service (Prometheus). Бизнес-счётчики — рядом с middleware.

import "github.com/prometheus/client_golang/prometheus"

var (
	// registerTotal — попытки регистрации (outcome: ok | email_taken | validation | error).
	registerTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_register_total",
		Help: "Registration attempts by outcome.",
	}, []string{"outcome"})

	// loginTotal — попытки входа (outcome: ok | invalid_credentials | locked | error).
	loginTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_login_total",
		Help: "Login attempts by outcome.",
	}, []string{"outcome"})

	// refreshTotal — попытки ротации refresh (outcome: ok | rejected).
	refreshTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_refresh_total",
		Help: "Refresh token rotations by outcome.",
	}, []string{"outcome"})
)

func init() {
	prometheus.MustRegister(registerTotal, loginTotal, refreshTotal)
}
