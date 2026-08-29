# HeyEV Backend POC

MQTT 5 backend for AWS IoT Core command/ACK experiments.

## Build

```bash
cd backend
go build .
```

## Run

```bash
export IOT_ENDPOINT="your-endpoint.iot.region.amazonaws.com"
go run .
```

Run from the `backend/` directory so certificate paths resolve.

On startup you will be guided through configuration with **y/n prompts** — no CLI flags to remember.

## Environment

| Variable | Required | Description |
|----------|----------|-------------|
| `IOT_ENDPOINT` | Yes | AWS IoT Core ATS endpoint hostname |

## Certificates

Relative to `backend/`:

- `../certs/backend/AmazonRootCA1.pem`
- `../certs/backend/46fe5a021abb1fa075d856403d9c3ce6ce183cfe15a5eb885e9d335b2db21849-certificate.pem.crt`
- `../certs/backend/46fe5a021abb1fa075d856403d9c3ce6ce183cfe15a5eb885e9d335b2db21849-private.pem.key`

## Interactive configuration (at startup)

```
========================================
HeyEV Backend — Configuration
========================================
Use QoS 1 instead of QoS 0? [n]:
Set RETAIN=true on command publish? [n]:
Enable DUP flag experiment logging? [n]:
Allow duplicate command publish (skip idempotency block)? [n]:
Enable auto-reconnect on connection loss? [y]:
Enable debug logging? [n]:
```

Press **Enter** to accept the default in `[brackets]`. Answer **y** or **n** for each option.

## Interactive flow (after config)

1. Enter device ID (or `+` for wildcard ACK subscribe)
2. If `+`, enter command device ID per command
3. Enter command, value, message expiry (seconds), optional request ID
4. Backend publishes to `heyev/v1/devices/{device_id}/commands`
5. Backend listens on `heyev/v1/devices/{device_id|+}/ack`

## Experiment examples

**QoS 1 test:** answer `y` to "Use QoS 1 instead of QoS 0?"

**Retain test:** answer `y` to "Set RETAIN=true on command publish?"

**Message expiry:** when prompted during command entry, enter e.g. `30`

**Duplicate command test:** answer `y` to "Allow duplicate command publish", then reuse the same request ID

**Duplicate ACK test:** run simulator in controlled mode and send two ACKs for the same request ID

**Reconnect test:** keep auto-reconnect enabled (default), disconnect network briefly

## MQTT 3.1.1

Not available. This POC uses `github.com/eclipse/paho.golang`, which is MQTT 5 only.

## Limitations

- Protocol DUP cannot be forced on outgoing publish with paho.golang
- `EXPIRED` state requires live AWS message expiry testing
- SUCCESS is only set if simulator sends `status: SUCCESS`
