# cmd-stream Stress Test (Go)

A stress-testing suite to validate [cmd-stream](https://github.com/cmd-stream/cmd-stream-go)
resilience under high concurrency and unstable network.

## Features

- Randomly sends `Echo`, `Stream`, and `Fail` Commands based on configurable 
  probabilities.
- Automatically verifies that received results match the expected output.
- Periodically restarts server and introduces downtime to simulate unstable 
  network.
- Tests the client's ability to reconnect and resume operations after server 
  restarts.
- Uses [circbrk](https://github.com/ymz-ncnk/circbrk-go) to provide 
  circuit-breaking capabilities.
- Has configurable long pauses to trigger and verify keepalive.
- Reports success rates, timeouts, network errors, and verification failures.

## Configuration

Configuration is managed via the `StressConfig` struct in [config.go](config.go). 
You can provide a YAML configuration file using the `-config` flag:

```bash
go run . -config my-config.yaml
```

If no config file is provided, the test uses default values. See [config.yaml.example](config.yaml.example) for all available options and their default values.

## Running the Stress Test

Simply execute:

```bash
go run .
```

The test will start 10 concurrent sessions (using 4 `cmd-stream` clients) and 
begin reporting statistics every 10 seconds. Use `Ctrl+C` to shut down the test 
gracefully.

## Summary Output

Here is an example of the last summary report from a 12-hour run:

```text
--- [STRESS TEST SUMMARY] ---
Total Commands: 3577792
  - Success:            1905640 (53.3%)     # Commands completed with verified results.
  - CB Blocked:         1647007 (46.0%)     # Commands prevented from sending by Circuit Breaker.
  - Keepalive Triggers: 35785               # Simulated idle periods to trigger keepalive.
  - Late Results:       4746                # Responses arrived after timeout.
  - Send Timeouts:      0 (0.0%)            # Timeout during Command send.
  - Result Timeouts:    6505 (0.2%)         # Timeout waiting for result.
  - Network Error:      18640 (0.5%)        # Connection issues (e.g. server down before CB trips).
  - Unexpected Err:     0 (0.0%)            # Uncategorized errors.
  - Verify Error:       0 (0.0%) [CRITICAL] # Received data mismatch.
-----------------------------
```

Any `Verify Error` greater than 0 indicates a bug in the library or the test 
itself and is considered a critical failure.

> [!NOTE]
> Interpreting the results:
>
> - QPS: The total Command count might seem low (~80 QPS) compared to raw 
>   benchmarks. This is due to artificial server delays, periodic downtimes, and 
>   client-side pauses used to simulate a realistic unstable environment.
> - CB Blocked: A high "CB Blocked" count is expected. When the Circuit Breaker 
>   opens during server downtime, sending sessions enter a "tight loop" and 
>   generate many blocked attempts until the system recovers.
> 
> The focus is on verifying stability and correctness under load, not maximum 
> throughput.
