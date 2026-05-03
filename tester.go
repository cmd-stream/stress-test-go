package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync/atomic"
	"time"

	cmdstream "github.com/cmd-stream/cmd-stream-go"
	cln "github.com/cmd-stream/cmd-stream-go/client"
	"github.com/cmd-stream/cmd-stream-go/core"
	ccln "github.com/cmd-stream/cmd-stream-go/core/cln"
	dcln "github.com/cmd-stream/cmd-stream-go/delegate/cln"
	grp "github.com/cmd-stream/cmd-stream-go/group"
	sndr "github.com/cmd-stream/cmd-stream-go/sender"
	hks "github.com/cmd-stream/cmd-stream-go/sender/hooks"
	"github.com/ymz-ncnk/circbrk-go"
)

const (
	echoCmd commandType = iota
	streamCmd
	failCmd
)

const (
	SendCmdTimeout    = time.Second
	WaitResultTimeout = MaxDelayDuration + 2*time.Millisecond
)

type commandType int

// StressTester manages the execution of concurrent test sessions and
// reports execution statistics.
type StressTester struct {
	sender sndr.Sender[struct{}]
	cfg    StressConfig

	successCount       int64
	cbBlockCount       int64
	sendTimeoutCount   int64
	resultTimeoutCount int64
	verifyErrCount     int64
	netErrCount        int64
	unexpectedErrCount int64
	keepaliveCount     int64
	lateResultCount    int64
}

func NewStressTester(addr string, codec cln.Codec[struct{}], cfg StressConfig) (
	*StressTester, error,
) {
	t := &StressTester{
		cfg: cfg,
	}
	sender, err := cmdstream.NewSender(addr, codec,
		sndr.WithClientsCount[struct{}](cfg.SenderClientsCount),
		sndr.WithSender(
			sndr.WithHooksFactory(
				hks.NewCircuitBreakerHooksFactory(
					circbrk.New(
						circbrk.WithWindowSize(cfg.CircuitBreakerWindowSize),
						circbrk.WithFailureRate(cfg.CircuitBreakerFailureRate),
						circbrk.WithOpenDuration(cfg.CircuitBreakerOpenDuration),
						circbrk.WithSuccessThreshold(cfg.CircuitBreakerSuccessThreshold),
						circbrk.WithChangeStateCallback(
							func(state circbrk.State) {
								fmt.Printf("--- Circuit Breaker state: %v ---\n", state)
							},
						),
					),
					hks.NoopHooksFactory[struct{}]{},
				),
			),
		),
		sndr.WithGroup(
			grp.WithReconnect[struct{}](),
			grp.WithClient[struct{}](
				cln.WithKeepalive(
					dcln.WithKeepaliveIntvl(cfg.KeepaliveIntvl),
					dcln.WithKeepaliveTime(cfg.KeepaliveTime),
				),
				cln.WithCore(
					ccln.WithUnexpectedResultCallback(
						func(seq core.Seq, result core.Result) {
							atomic.AddInt64(&t.lateResultCount, 1)
						},
					),
				),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	t.sender = sender
	go t.reportLoop()
	return t, nil
}

func (t *StressTester) RunSession(ctx context.Context, id int) {
	fmt.Printf("Session %d starting\n", id)
	ctx = context.WithValue(ctx, "sessionID", id)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			t.sendRandomCommand(ctx)
		}
	}
}

func (t *StressTester) sendRandomCommand(ctx context.Context) {
	ownCtx, cancel := context.WithTimeout(ctx, WaitResultTimeout)
	defer cancel()

	switch t.nextCommandType() {
	case failCmd:
		t.sendFailCmd(ownCtx)
	case echoCmd:
		t.sendEchoCmd(ownCtx)
	case streamCmd:
		t.sendStreamCmd(ownCtx)
	}

	pause := t.nextPause()
	if pause > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(pause):
		}
	}
}

func (t *StressTester) sendFailCmd(ctx context.Context) {
	var (
		sessionID = ctx.Value("sessionID").(int)
		deadline  = time.Now().Add(SendCmdTimeout)
	)
	_, err := t.sender.SendWithDeadline(ctx, deadline, FailCmd{})
	t.updateStats(sessionID, "FailCmd", err, false)
}

func (t *StressTester) sendEchoCmd(ctx context.Context) {
	var (
		sessionID    = ctx.Value("sessionID").(int)
		payload      = fmt.Sprintf("hello-%d", rand.Intn(1000))
		verifyFailed = false
		deadline     = time.Now().Add(SendCmdTimeout)
	)
	res, err := t.sender.SendWithDeadline(ctx, deadline, EchoCmd(payload))
	if err == nil && string(res.(EchoResult)) != payload {
		fmt.Printf("Session %d: Echo verification failed: expected %s, got %s\n", sessionID, payload, res)
		verifyFailed = true
	}
	t.updateStats(sessionID, "Echo", err, verifyFailed)
}

func (t *StressTester) sendStreamCmd(ctx context.Context) {
	var (
		sessionID    = ctx.Value("sessionID").(int)
		initNum      = rand.Intn(1000)
		count        = rand.Intn(t.cfg.MaxStreamResults) + 1
		received     = 0
		verifyFailed = false
		cmd          = StreamCmd{InitNum: initNum, ResultsCount: count}
		deadline     = time.Now().Add(SendCmdTimeout)
	)
	err := t.sender.SendMultiWithDeadline(ctx, deadline, cmd, count, sndr.ResultHandlerFn(
		func(result core.Result, err error) error {
			if err != nil {
				return err
			}
			var (
				res      = result.(StreamResult)
				val      = res.Value
				expected = initNum + received
			)
			if val != expected {
				fmt.Printf("Session %d: Stream verification failed: expected %d, got %d\n", sessionID, expected, val)
				verifyFailed = true
			}
			received++
			return nil
		},
	))

	t.updateStats(sessionID, "Stream", err, verifyFailed)
}

func (t *StressTester) reportLoop() {
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		var (
			success       = atomic.LoadInt64(&t.successCount)
			cbBlock       = atomic.LoadInt64(&t.cbBlockCount)
			sendTimeout   = atomic.LoadInt64(&t.sendTimeoutCount)
			resultTimeout = atomic.LoadInt64(&t.resultTimeoutCount)
			verifyErr     = atomic.LoadInt64(&t.verifyErrCount)
			netErr        = atomic.LoadInt64(&t.netErrCount)
			unexpected    = atomic.LoadInt64(&t.unexpectedErrCount)
			keepalive     = atomic.LoadInt64(&t.keepaliveCount)
			lateResults   = atomic.LoadInt64(&t.lateResultCount)
			total         = success + cbBlock + sendTimeout + resultTimeout + verifyErr + netErr + unexpected
		)
		if total == 0 {
			continue
		}

		fmt.Printf("\n--- [STRESS TEST SUMMARY] ---\n")
		fmt.Printf("Total Commands: %d\n", total)
		fmt.Printf("  - Success:            %d (%.1f%%)\n", success, float64(success)/float64(total)*100)
		fmt.Printf("  - CB Blocked:         %d (%.1f%%)\n", cbBlock, float64(cbBlock)/float64(total)*100)
		fmt.Printf("  - Keepalive Triggers: %d\n", keepalive)
		fmt.Printf("  - Late Results:       %d\n", lateResults)
		fmt.Printf("  - Send Timeouts:      %d (%.1f%%)\n", sendTimeout, float64(sendTimeout)/float64(total)*100)
		fmt.Printf("  - Result Timeouts:    %d (%.1f%%)\n", resultTimeout, float64(resultTimeout)/float64(total)*100)
		fmt.Printf("  - Network Error:      %d (%.1f%%)\n", netErr, float64(netErr)/float64(total)*100)
		fmt.Printf("  - Unexpected Error:   %d (%.1f%%)\n", unexpected, float64(unexpected)/float64(total)*100)
		fmt.Printf("  - Verify Error:       %d (%.1f%%) [CRITICAL]\n", verifyErr, float64(verifyErr)/float64(total)*100)
		fmt.Printf("-----------------------------\n\n")
	}
}

func (t *StressTester) nextCommandType() commandType {
	r := rand.Float64()
	if r < t.cfg.FailProb {
		return failCmd
	}
	if r < t.cfg.FailProb+t.cfg.EchoProb {
		return echoCmd
	}
	return streamCmd
}

func (t *StressTester) nextPause() time.Duration {
	r := rand.Float64()
	if r < t.cfg.NoPauseProb {
		return 0
	}
	if r < t.cfg.NoPauseProb+t.cfg.WorkloadPauseProb {
		return time.Duration(rand.Int63n(int64(t.cfg.WorkloadPauseMax)))
	}
	// Rarely, pause long enough to trigger keepalive.
	atomic.AddInt64(&t.keepaliveCount, 1)
	return t.cfg.KeepaliveTime + time.Duration(rand.Int63n(int64(t.cfg.KeepalivePauseMax)))
}

func (t *StressTester) updateStats(sessionID int, cmdName string, err error,
	verifyFailed bool) {
	if verifyFailed {
		atomic.AddInt64(&t.verifyErrCount, 1)
		return
	}
	if err == nil {
		atomic.AddInt64(&t.successCount, 1)
		return
	}
	if errors.Is(err, hks.ErrNotAllowed) {
		atomic.AddInt64(&t.cbBlockCount, 1)
		return
	}

	// 1. Result Timeout.
	if errors.Is(err, sndr.ErrTimeout) {
		atomic.AddInt64(&t.resultTimeoutCount, 1)
		return
	}

	// 2. Send Timeout.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		// If network timeout occurs during the waiting for tje result, the client
		// will close.
		atomic.AddInt64(&t.sendTimeoutCount, 1)
		return
	}

	// 3. Known Network Errors.
	if isNetworkError(err) {
		atomic.AddInt64(&t.netErrCount, 1)
		return
	}
	fmt.Printf("Session %d: %s UNEXPECTED error: %v\n", sessionID, cmdName, err)
	atomic.AddInt64(&t.unexpectedErrCount, 1)
}

func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	return errors.As(err, &netErr) || errors.Is(err, io.EOF)
}
