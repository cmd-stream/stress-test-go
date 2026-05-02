package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	cmdstream "github.com/cmd-stream/cmd-stream-go"
	csrv "github.com/cmd-stream/cmd-stream-go/core/srv"
	srv "github.com/cmd-stream/cmd-stream-go/server"
)

// RestartableServer wraps a cmd-stream server and provides mechanisms
// to simulate service instability through periodic restarts.
type RestartableServer struct {
	addr         string
	workersCount int
	codec        srv.Codec[struct{}]
	srv          *csrv.Server
	mu           sync.Mutex
}

func NewRestartableServer(addr string, workersCount int,
	codec srv.Codec[struct{}],
) *RestartableServer {
	return &RestartableServer{
		addr:         addr,
		workersCount: workersCount,
		codec:        codec,
	}
}

func (s *RestartableServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	server, err := cmdstream.NewServer(struct{}{}, s.codec,
		srv.WithCore(
			csrv.WithWorkersCount(s.workersCount),
		),
	)
	if err != nil {
		return err
	}
	s.srv = server
	go func() {
		if err := s.srv.ListenAndServe(s.addr); err != nil {
			if !errors.Is(err, csrv.ErrClosed) && !errors.Is(err, csrv.ErrShutdown) {
				panic(err)
			}
		}
	}()
	return nil
}

func (s *RestartableServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != nil {
		err := s.srv.Close()
		s.srv = nil
		return err
	}
	return nil
}

func (s *RestartableServer) LifecycleLoop(intervalMax, downtimeMax time.Duration) {
	for {
		workInterval := time.Duration(rand.Int63n(int64(intervalMax)))
		time.Sleep(workInterval)

		fmt.Println("--- Lifecycle: Stopping server ---")
		if err := s.Stop(); err != nil {
			fmt.Printf("Error stopping server: %v\n", err)
		}

		downtime := time.Duration(rand.Int63n(int64(downtimeMax)))
		fmt.Printf("--- Lifecycle: Server downtime for %v ---\n", downtime)
		time.Sleep(downtime)

		fmt.Println("--- Lifecycle: Restarting server ---")
		if err := s.Start(); err != nil {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}
}
