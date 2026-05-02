package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cdcjson "github.com/cmd-stream/codec-json-go"
)

func main() {
	const addr = "127.0.0.1:9000"

	// 1. Setup JSON codec.
	var (
		reg = cdcjson.NewRegistry(
			cdcjson.WithCmd[struct{}, EchoCmd](),
			cdcjson.WithCmd[struct{}, StreamCmd](),
			cdcjson.WithCmd[struct{}, FailCmd](),
			cdcjson.WithResult[struct{}, EchoResult](),
			cdcjson.WithResult[struct{}, StreamResult](),
		)
		serverCodec = cdcjson.NewServerCodecWith(reg)
		clientCodec = cdcjson.NewClientCodecWith(reg)
	)

	// 2. Setup stress config.
	cfg := StressConfig{
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

	// 3. Start restartable server.
	server := NewRestartableServer(addr, cfg.ServerWorkersCount, serverCodec)
	if err := server.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	go server.LifecycleLoop(cfg.ServerWorkIntervalMax, cfg.ServerDowntimeMax)

	// 4. Start stress tester.
	time.Sleep(100 * time.Millisecond)
	tester, err := NewStressTester(addr, clientCodec, cfg)
	if err != nil {
		fmt.Printf("Failed to create stress tester: %v\n", err)
		server.Stop()
		time.Sleep(1 * time.Second)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Spawn sessions.
	for i := range cfg.SessionsCount {
		go tester.RunSession(ctx, i)
	}

	// 5. Wait for interrupt.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("Shutting down stress test...")
	cancel()
	server.Stop()
	time.Sleep(1 * time.Second)
}
