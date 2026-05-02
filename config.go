package main

import "time"

type StressConfig struct {
	// Probabilities of choosing a Command type.
	FailProb float64 // Probability of sending FailCmd.
	EchoProb float64 // Probability of sending EchoCmd.
	// StreamProb is calculated as 1 - FailProb - EchoProb.

	MaxStreamResults int // Maximum results count for StreamCmd.

	// Probabilities of pausing between Commands.
	NoPauseProb       float64 // Probability of no pause.
	WorkloadPauseProb float64 // Probability of a short pause between sending Commands.
	// Keepalive pause probability is calculated as 1 - NoPauseProb - WorkloadPauseProb.

	// Pause durations.
	WorkloadPauseMax  time.Duration // Maximum duration of a short pause between sending Commands.
	KeepalivePauseMax time.Duration // Maximum period of time in keepalive mode.

	// Circuit Breaker settings.
	CircuitBreakerWindowSize       int           // Number of requests in the sliding window.
	CircuitBreakerFailureRate      float64       // Failure rate threshold to open the circuit.
	CircuitBreakerOpenDuration     time.Duration // Time to wait before transitioning to HalfOpen.
	CircuitBreakerSuccessThreshold int           // Number of successes in HalfOpen to Close the circuit.

	// Keepalive settings.
	KeepaliveIntvl time.Duration // Interval between keepalive pings.
	KeepaliveTime  time.Duration // Time without activity before sending a ping.

	// Concurrency settings.
	SessionsCount      int // Number of concurrent sessions.
	SenderClientsCount int // Number of clients in the sender pool.
	ServerWorkersCount int // Number of server workers.

	// Server lifecycle settings.
	ServerWorkIntervalMax time.Duration // Maximum time the server stays up.
	ServerDowntimeMax     time.Duration // Maximum time the server stays down.
}
