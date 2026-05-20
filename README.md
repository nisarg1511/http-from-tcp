# Custom HTTP/1.1 Server From Scratch (TCP Wrapper)

A lightweight, HTTP server built directly on top of raw Layer 4 TCP sockets in Go, without relying on the `net/http` standard library package. This project demonstrates low-level network protocol manipulation, stateful byte-buffer parsing, and connection management.

## 🚀 Features

- **Protocol-Aware Routing**: Dynamically inspects the incoming request line and splits processing between `HTTP/1.0` (non-persistent) and `HTTP/1.1` (persistent Keep-Alive) execution paths.
- **Connection Persistence (Keep-Alive)**: Reuses a single underlying TCP connection for sequential HTTP/1.1 requests, optimizing throughput and handshake latency.
- **Resource Protection & Timeout Safety**: Implements explicit context-driven deadlines (`SetDeadline`) to automatically reap idle or orphaned sockets and protect against connection leaks.
- **Deterministic Header Draining**: Safely drains residual network buffer streams to prevent downstream data frame pollution and runtime slice panics.

## 🧱 Architecture Flow

```text
       [ Incoming TCP Connection ]
                   │
                   ▼
         handleConnection(conn)
                   │
         (Parse Request Line)
                   │
         ┌─────────┴─────────┐
         ▼                   ▼
    [ HTTP/1.0 ]        [ HTTP/1.1 ]
  Non-Persistent         Persistent
  (Drain -> Close)     (Keep-Alive Loop)
                             │
                             ▼
                     (5s Idle Timeout)
```

## 💻 Getting Started

### Prerequisites

- Go 1.22 or higher installed.

### Running the Server

1. Clone the repository and navigate into the directory.
2. Run the application:
   ```bash
   go run main.go
   ```
3. The server will spin up and start listening on `http://localhost:8080`.

## 🛠️ Verification & Testing

You can trace network state transitions using `curl` with verbose logging enabled:

### Test Non-Persistent Pipeline (HTTP/1.0)

```bash
curl -v --http1.0 http://localhost:8080/resource
```

_Observe that the server logs `[Routing to NON-PERSISTENT]` and explicitly transmits a `Connection: close` header before cleanly shutting down the socket connection._

### Test Persistent Pipeline (HTTP/1.1 Keep-Alive)

```bash
curl -v http://localhost:8080/resource
```

_Observe that the server logs `[Routing to PERSISTENT]`. The TCP socket remains attached and receptive to further frames until it explicitly hits the 5-second inactivity timeout._

## 🔬 Key Engineering Insights & Discoveries

During development and stress-testing using live browser refreshes, I uncovered two critical browser-to-server behaviors:

1. **The Criticality of Header Draining in Persistent Connections**:
   Omitting the header draining routine allows lingering HTTP headers (like `Host`, `User-Agent`) to pollute the Layer 4 TCP buffer. On the subsequent iteration of the persistent loop, the parser misinterprets these residual headers as a new HTTP Request Line. This causes a structural validation failure (`len(parts) != 3`) and violently tears down the connection.
2. **Implicit Browser Retries & Idempotency**:
   When the persistent connection is abruptly severed due to the socket validation crash described above, modern web browsers execute an immediate, silent retry. Because a browser refresh fires an _idempotent_ `GET` request, the client automatically spins up a brand new TCP connection behind the scenes to recover seamlessly. This manifests as rapid, subsequent connection triggers in the server console logging.
