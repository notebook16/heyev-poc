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
Command delivery mode:
  1) Option A — Retain + Message Expiry (late subscribe on topic)
  2) Option B — Session queue + Message Expiry (device offline queue)
Select delivery mode [2]:
Use QoS 1 instead of QoS 0? [y for Option B]:
Set RETAIN=true on command publish? [n, skipped for Option B]:
Enable DUP flag experiment logging? [n]:
Allow duplicate command publish (skip idempotency block)? [n]:
Enable auto-reconnect on connection loss? [y]:
Session expiry in seconds (0 = session ends on disconnect) [900 for Option B]:
Use persistent MQTT session (resume after reconnect)? [y for Option B]:
Enable debug logging? [n]:
```

Press **Enter** to accept the default in `[brackets]`. Answer **y** or **n** for each option.

## Command delivery modes

### Option A — Retain + Message Expiry

Best for late/first-connect scenarios where the device may subscribe after the command is published.

- Backend may set **retain=true** on command publish
- Set **Message Expiry** per command at runtime
- Device receives the last retained command when it subscribes (if not expired)

**Test procedure:**

1. Start backend → select **Option A**, enable retain if desired
2. Publish a command with message expiry (e.g. 60s)
3. Start simulator **after** the publish
4. Simulator should receive the retained command on subscribe

### Option B — Session queue + Message Expiry (recommended for production)

Best for offline device delivery via the broker session queue.

- **QoS 1** required on backend publish and simulator subscribe
- **retain=false** (forced)
- Simulator uses **persistent session** + **session expiry > 0** + stable **Client ID**
- Set **Message Expiry** per command at runtime
- Commands queue while the device is offline and deliver on reconnect

**Test procedure:**

1. Start simulator first → select **Option B**, QoS 1, persistent session, session expiry 900s
2. Stop simulator (Ctrl+C) or disconnect network
3. Start backend → select **Option B**, QoS 1
4. Publish command with message expiry (e.g. 120s)
5. Restart simulator with the **same Client ID**
6. Watch for `[SESSION] SessionPresent (CONNACK): true` and queued command delivery

## Interactive flow (after config)

1. Enter device ID (or `+` for wildcard ACK subscribe)
2. If `+`, enter command device ID per command
3. Enter command, value, message expiry (seconds), optional request ID
4. Backend publishes to `heyev/v1/devices/{device_id}/commands`
5. Backend listens on `heyev/v1/devices/{device_id|+}/ack`

## Experiment examples

**QoS 1 test:** answer `y` to "Use QoS 1 instead of QoS 0?"

**Retain test (Option A only):** answer `y` to "Set RETAIN=true on command publish?"

**Message expiry:** when prompted during command entry, enter e.g. `30`

**Duplicate command test:** answer `y` to "Allow duplicate command publish", then reuse the same request ID

**Duplicate ACK test:** run simulator in controlled mode and send two ACKs for the same request ID

**Reconnect test:** keep auto-reconnect enabled (default), disconnect network briefly

**Session queue test (Option B):** see Option B test procedure above

## MQTT 3.1.1

Not available. This POC uses `github.com/eclipse/paho.golang`, which is MQTT 5 only.

## Limitations

- Protocol DUP cannot be forced on outgoing publish with paho.golang
- `EXPIRED` state requires live AWS message expiry testing
- SUCCESS is only set if simulator sends `status: SUCCESS`
- Option B offline queue behavior requires live AWS IoT Core testing
