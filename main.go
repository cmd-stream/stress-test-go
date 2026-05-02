package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cdcjson "github.com/cmd-stream/codec-json-go"
)

func main() {
	configPath := flag.String("config", "", "Path to YAML configuration file")
	flag.Parse()

	// 1. Setup stress config.
	cfg, err := LoadConfig(*configPath)
	if err != nil && !os.IsNotExist(err) {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	// 2. Setup JSON codec.
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

	// 3. Start restartable server.
	server := NewRestartableServer(cfg.Address, cfg.ServerWorkersCount, serverCodec)
	if err := server.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	go server.LifecycleLoop(cfg.ServerWorkIntervalMax, cfg.ServerDowntimeMax)

	// 4. Start stress tester.
	time.Sleep(100 * time.Millisecond)
	tester, err := NewStressTester(cfg.Address, clientCodec, cfg)
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
