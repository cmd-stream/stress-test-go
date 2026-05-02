package main

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type StressConfig struct {
	// Address of the stress test server.
	Address string `yaml:"address"`

	// Probabilities of choosing a Command type.
	FailProb float64 `yaml:"fail_prob"` // Probability of sending FailCmd.
	EchoProb float64 `yaml:"echo_prob"` // Probability of sending EchoCmd.
	// StreamProb is calculated as 1 - FailProb - EchoProb.

	MaxStreamResults int `yaml:"max_stream_results"` // Maximum results count for StreamCmd.

	// Probabilities of pausing between Commands.
	NoPauseProb       float64 `yaml:"no_pause_prob"`       // Probability of no pause.
	WorkloadPauseProb float64 `yaml:"workload_pause_prob"` // Probability of a short pause between sending Commands.
	// Keepalive pause probability is calculated as 1 - NoPauseProb - WorkloadPauseProb.

	// Pause durations.
	WorkloadPauseMax  time.Duration `yaml:"workload_pause_max"`  // Maximum duration of a short pause between sending Commands.
	KeepalivePauseMax time.Duration `yaml:"keepalive_pause_max"` // Maximum period of time in keepalive mode.

	// Circuit Breaker settings.
	CircuitBreakerWindowSize       int           `yaml:"cb_window_size"`       // Number of requests in the sliding window.
	CircuitBreakerFailureRate      float64       `yaml:"cb_failure_rate"`      // Failure rate threshold to open the circuit.
	CircuitBreakerOpenDuration     time.Duration `yaml:"cb_open_duration"`     // Time to wait before transitioning to HalfOpen.
	CircuitBreakerSuccessThreshold int           `yaml:"cb_success_threshold"` // Number of successes in HalfOpen to Close the circuit.

	// Keepalive settings.
	KeepaliveIntvl time.Duration `yaml:"keepalive_intvl"` // Interval between keepalive pings.
	KeepaliveTime  time.Duration `yaml:"keepalive_time"`  // Time without activity before sending a ping.

	// Concurrency settings.
	SessionsCount      int `yaml:"sessions_count"`       // Number of concurrent sessions.
	SenderClientsCount int `yaml:"sender_clients_count"` // Number of clients in the sender pool.
	ServerWorkersCount int `yaml:"server_workers_count"` // Number of server workers.

	// Server lifecycle settings.
	ServerWorkIntervalMax time.Duration `yaml:"server_work_interval_max"` // Maximum time the server stays up.
	ServerDowntimeMax     time.Duration `yaml:"server_downtime_max"`      // Maximum time the server stays down.
}

// DefaultConfig returns the default stress test configuration.
func DefaultConfig() StressConfig {
	return StressConfig{
		Address: "127.0.0.1:9000",

		FailProb:          0.0001,
		EchoProb:          0.4999,
		MaxStreamResults:  5,
		NoPauseProb:       0.8,
		WorkloadPauseProb: 0.19,
		WorkloadPauseMax:  500 * time.Millisecond,
		KeepalivePauseMax: 6 * time.Second,

		CircuitBreakerWindowSize:       10,
		CircuitBreakerFailureRate:      0.5,
		CircuitBreakerOpenDuration:     3 * time.Second,
		CircuitBreakerSuccessThreshold: 2,

		KeepaliveIntvl: time.Second,
		KeepaliveTime:  2 * time.Second,

		SessionsCount:      10,
		SenderClientsCount: 4,
		ServerWorkersCount: 20,

		ServerWorkIntervalMax: 20 * time.Second,
		ServerDowntimeMax:     5 * time.Second,
	}
}

// LoadConfig loads configuration from a YAML file.
func LoadConfig(path string) (StressConfig, error) {
	cfg := DefaultConfig()
	if path == "" {
		return cfg, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&cfg)
	return cfg, err
}
