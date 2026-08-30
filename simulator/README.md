# IoT Simulator POC

MQTT 5 device simulator for AWS IoT Core command/ACK experiments.

## Build

```bash
cd simulator
go build .
```

## Run

```bash
export IOT_ENDPOINT="your-endpoint.iot.region.amazonaws.com"
go run .
```

Run from the `simulator/` directory so certificate paths resolve.

On startup you will be guided through configuration with **numbered and y/n prompts** — no CLI flags to remember.

## Environment

| Variable | Required | Description |
|----------|----------|-------------|
| `IOT_ENDPOINT` | Yes | AWS IoT Core ATS endpoint hostname |

## Certificates

Relative to `simulator/`:

- `../certs/simulator/AmazonRootCA1.pem`
- `../certs/simulator/a75bd3d3cf1ed6d465c3114ee171b232cef5cb6e5e0eb9a01dd83034001d9879-certificate.pem.crt`
- `../certs/simulator/a75bd3d3cf1ed6d465c3114ee171b232cef5cb6e5e0eb9a01dd83034001d9879-private.pem.key`

## Interactive configuration (at startup)

```
========================================
IoT Simulator — Configuration
========================================
Simulator mode:
  1) Autonomous — auto ACK on every command
  2) Controlled  — you choose when/how to ACK
Select mode [1]:
Command delivery mode (must match backend):
  1) Option A — Retain + Message Expiry
  2) Option B — Session queue + Message Expiry
Select delivery mode [2]:
Client ID (blank = default) [iot-simulator-poc]:
Use QoS 1 for command subscription and ACK publish? [y for Option B]:
Set RETAIN=true on ACK publish? [n]:
Log incoming MQTT DUP flag on commands? [n]:
Enable auto-reconnect on connection loss? [y]:
Session expiry in seconds (0 = session ends on disconnect) [900 for Option B]:
Use persistent MQTT session (offline command queue)? [y for Option B]:
Enable debug logging? [n]:
```

Press **Enter** to accept defaults. Answer **y** or **n** for yes/no options.

## Command delivery modes

### Option A — Retain + Message Expiry

Match backend Option A. Subscribe with QoS 0 or 1. When connecting after a retained command was published, the broker delivers the retained message on subscribe.

### Option B — Session queue + Message Expiry

Match backend Option B for production-style offline delivery.

Requirements (enforced at startup):

- **QoS 1** command subscription
- **Persistent session** enabled
- **Session expiry > 0** (default 900s)
- Stable **Client ID** across reconnects

When the simulator disconnects, commands published by the backend queue at the broker. On reconnect with the same Client ID, watch for:

```
[SESSION] SessionPresent (CONNACK): true
[SESSION] Option B: broker may deliver queued commands after reconnect
```

## Modes

### Autonomous (default — select 1)

- Subscribes to `heyev/v1/devices/+/commands`
- Parses device ID from topic
- Auto-publishes ACK to `heyev/v1/devices/{device_id}/ack`

### Controlled (select 2)

- Prompt for command subscribe target (`device_id` or `+`)
- Shows each command with duplicate status
- You choose: accept/ignore, send ACK or skip, ACK topic

## Experiment examples

**QoS 1 ACK:** answer `y` to "Use QoS 1 for command subscription and ACK publish?"

**Retained ACK:** answer `y` to "Set RETAIN=true on ACK publish?"

**Duplicate ACK test:** select controlled mode, accept first command and ACK, then choose whether to ACK duplicate

**Reconnect test:** keep auto-reconnect enabled (default)

**Option B offline queue test:**

1. Start simulator with Option B (QoS 1, persistent session, session expiry 900)
2. Stop simulator
3. Publish command from backend (Option B, QoS 1, message expiry e.g. 120)
4. Restart simulator with the **same Client ID**
5. Confirm queued command is delivered

## MQTT 3.1.1

Not available with `github.com/eclipse/paho.golang` (MQTT 5 only).

## Limitations

- Cannot force protocol DUP on outgoing ACK publish
- Autonomous mode always ACKs immediately
- Controlled mode requires terminal interaction per command
- Option B session queue behavior requires live AWS IoT Core testing
