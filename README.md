# Go Load Balancer

A simple yet robust HTTP load balancer written in Go that distributes incoming requests across multiple backend servers using round-robin algorithm with health checking capabilities.

## Features

- **Round-Robin Load Balancing**: Distributes requests evenly across available backend servers
- **Health Checking**: Automatically monitors backend server health and removes unhealthy servers
- **Retry Logic**: Implements retry mechanism for failed requests (up to 3 retries)
- **Circuit Breaker**: Marks servers as down after repeated failures
- **Concurrent Safe**: Thread-safe implementation using atomic operations and mutexes
- **Reverse Proxy**: Uses Go's built-in reverse proxy for efficient request forwarding

## Architecture

The load balancer consists of several key components:

- **Backend**: Represents a backend server with URL, health status, and reverse proxy
- **ServerPool**: Manages a collection of backend servers and implements round-robin selection
- **Health Checker**: Periodically checks backend server availability
- **Load Balancer**: Main handler that routes requests to healthy backends

## Prerequisites

- Go 1.19 or later
- Multiple backend servers to load balance (optional for testing)

## Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd go-load-balancer
```

2. Build the application:
```bash
go build -o load-balancer load-balancer.go
```

## Usage

### Basic Usage

1. **Start the load balancer**:
```bash
./load-balancer
```

The load balancer will start on port 3000 and attempt to connect to the following backend servers:
- `http://localhost:8081`
- `http://localhost:8082`
- `http://localhost:8083`

2. **Send requests to the load balancer**:
```bash
curl http://localhost:3000
```

### Testing with Backend Servers

To test the load balancer with actual backend servers, you can create simple HTTP servers:

1. **Create test backend servers** (in separate terminals):

```bash
# Terminal 1 - Backend Server 1
python3 -m http.server 8081

# Terminal 2 - Backend Server 2
python3 -m http.server 8082

# Terminal 3 - Backend Server 3
python3 -m http.server 8083
```

2. **Start the load balancer**:
```bash
./load-balancer
```

3. **Test load balancing**:
```bash
# Send multiple requests to see round-robin distribution
for i in {1..10}; do
  echo "Request $i:"
  curl -s http://localhost:3000 | head -1
  echo
done
```

## Configuration

### Modifying Backend Servers

To change the backend servers, edit the `targets` slice in the `main()` function:

```go
targets := []string{
    "http://localhost:8081",
    "http://localhost:8082",
    "http://localhost:8083",
    "http://your-server:8084",  // Add your server
}
```

### Adjusting Health Check Interval

Modify the health check interval in the `healthCheck()` function:

```go
t := time.NewTicker(time.Second * 20)  // Change 20 to desired seconds
```

### Changing Load Balancer Port

Update the port in the `main()` function:

```go
port := 3000  // Change to your desired port
```

## How It Works

### Load Balancing Algorithm

The load balancer uses a **round-robin** algorithm:

1. Maintains a current index counter
2. For each request, increments the counter atomically
3. Selects the next available backend server
4. Skips unhealthy servers automatically

### Health Checking

- **Interval**: Every 20 seconds (configurable)
- **Method**: TCP connection test with 2-second timeout
- **Behavior**: Marks servers as down if unreachable, up if reachable

### Retry Logic

- **Max Retries**: 3 attempts per request
- **Retry Delay**: 10 milliseconds between retries
- **Failure Handling**: After 3 failed retries, marks server as down and tries next server

### Request Flow

```
Client Request → Load Balancer → Select Healthy Backend → Forward Request
                     ↓
              If Backend Fails → Retry (up to 3 times) → Try Next Backend
                     ↓
              If All Backends Fail → Return 503 Service Unavailable
```

## Monitoring and Logging

The load balancer provides detailed logging:

- **Startup**: Server address and configuration
- **Health Checks**: Backend server status updates
- **Request Failures**: Error details and retry attempts
- **Server Status**: Up/down status for each backend

Example log output:
```
Load balancer started at :3000
Start healthcheck
http://localhost:8081 [up]
http://localhost:8082 [down]
http://localhost:8083 [up]
Healthcheck completed
```

## Error Handling

The load balancer handles various error scenarios:

- **Backend Unavailable**: Automatically retries with next server
- **All Backends Down**: Returns HTTP 503 Service Unavailable
- **Network Timeouts**: Configurable timeout for health checks
- **Concurrent Access**: Thread-safe operations using mutexes

## Performance Considerations

- **Atomic Operations**: Uses atomic counters for thread-safe round-robin
- **Connection Pooling**: Leverages Go's built-in HTTP client connection pooling
- **Efficient Health Checks**: Lightweight TCP connection tests
- **Memory Efficient**: Minimal memory footprint with simple data structures

## Development

### Building from Source

```bash
# Build for current platform
go build -o load-balancer load-balancer.go

# Build for different platforms
GOOS=linux GOARCH=amd64 go build -o load-balancer-linux load-balancer.go
GOOS=windows GOARCH=amd64 go build -o load-balancer.exe load-balancer.go
```

### Running Tests

```bash
# Run any tests (if available)
go test ./...
```

### Code Structure

```
load-balancer.go
├── Backend struct          # Backend server representation
├── ServerPool struct       # Backend collection and round-robin logic
├── main()                  # Application entry point
├── lb()                    # Load balancer handler
├── healthCheck()           # Health monitoring goroutine
└── Helper functions        # Utility functions for health checks and retries
```

## Troubleshooting

### Common Issues

1. **"Service not available" error**:
   - Check if backend servers are running
   - Verify backend URLs in configuration
   - Check health check logs

2. **Backend servers not being detected**:
   - Ensure servers are listening on correct ports
   - Check firewall settings
   - Verify URL format (include http://)

3. **High CPU usage**:
   - Adjust health check interval
   - Check for backend server issues causing frequent health check failures

### Debug Mode

Add debug logging by modifying the log level:

```go
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is open source and available under the [MIT License](LICENSE).

## Future Enhancements

Potential improvements for future versions:

- [ ] Weighted round-robin algorithm
- [ ] Least connections algorithm
- [ ] Configuration file support
- [ ] Metrics and monitoring endpoints
- [ ] SSL/TLS termination
- [ ] Sticky sessions
- [ ] Rate limiting
- [ ] Web dashboard for monitoring
