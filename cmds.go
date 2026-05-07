package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/cmd-stream/cmd-stream-go/core"
)

const MaxDelayDuration = 100 * time.Millisecond

// EchoCmd echoes the payload back to the client.
type EchoCmd string

func (c EchoCmd) Exec(ctx context.Context, receiver struct{}, proxy core.Proxy) error {
	randomDelay()
	_, err := proxy.Send(EchoResult(c))
	return err
}

// StreamCmd sends a sequence of results back to the client.
type StreamCmd struct {
	InitNum      int
	ResultsCount int
}

func (c StreamCmd) Exec(ctx context.Context, receiver struct{}, proxy core.Proxy) error {
	if c.ResultsCount == 0 {
		fmt.Printf("PANIC PREVENTED: ResultsCount is 0. InitNum: %d\n", c.InitNum)
	}
	ms := rand.Int63n(int64(MaxDelayDuration)) / int64(c.ResultsCount)
	for i := 0; i < c.ResultsCount; i++ {
		delay(ms)
		val := c.InitNum + i
		last := i == c.ResultsCount-1
		_, err := proxy.Send(StreamResult{Value: val, Last: last})
		if err != nil {
			return err
		}
	}
	return nil
}

// FailCmd intentionally returns an error, causing the server to drop the client
// connection.
type FailCmd struct{}

func (c FailCmd) Exec(ctx context.Context, receiver struct{}, proxy core.Proxy) error {
	randomDelay()
	fmt.Println("--- FailCmd: going to drop the connection ---")
	return errors.New("intentional failure")
}

func randomDelay() {
	ms := rand.Int63n(int64(MaxDelayDuration))
	delay(ms)
}

func delay(ms int64) {
	time.Sleep(time.Duration(ms))
}
