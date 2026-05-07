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

## Running the Stress Test

Simply execute:

```bash
# Run with default settings
go run .

# Or with custom configuration
go run . -config my-config.yaml
```

The test will start 10 concurrent sessions (using 4 `cmd-stream` clients) and 
begin reporting statistics every 10 seconds. To stop, use `Ctrl+C`.

In [config.yaml.example](config.yaml.example) you can find all available 
configuration options.

## Summary Output

Here is an example of the last summary report from a 12-hour run:

```text

--- [STRESS TEST SUMMARY] ---
Total Commands: 4069635
  - Success:            2197245 (54.0%)
  - CB Blocked:         1844078 (45.3%)
  - Keepalive Triggers: 40710
  - Late Results:       4912
  - Send Timeouts:      0 (0.0%)
  - Result Timeouts:    7367 (0.2%)
  - Network Error:      20945 (0.5%)
  - Unexpected Error:   0 (0.0%)
  - Verify Error:       0 (0.0%) [CRITICAL]
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
> - CB Blocked: A high `CB Blocked` count is expected. When the Circuit Breaker 
>   opens during server downtime, sending sessions enter a "tight loop" and 
>   generate many blocked attempts until the system recovers.
> 
> The focus is on verifying stability and correctness under load, not maximum 
> throughput.
