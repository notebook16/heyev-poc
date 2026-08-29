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
Use QoS 1 instead of QoS 0 for ACK publish? [n]:
Set RETAIN=true on ACK publish? [n]:
Log incoming MQTT DUP flag on commands? [n]:
Enable auto-reconnect on connection loss? [y]:
Enable debug logging? [n]:
```

Press **Enter** to accept defaults. Answer **y** or **n** for yes/no options.

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

**QoS 1 ACK:** answer `y` to "Use QoS 1 instead of QoS 0?"

**Retained ACK:** answer `y` to "Set RETAIN=true on ACK publish?"

**Duplicate ACK test:** select controlled mode, accept first command and ACK, then choose whether to ACK duplicate

**Reconnect test:** keep auto-reconnect enabled (default)

## MQTT 3.1.1

Not available with `github.com/eclipse/paho.golang` (MQTT 5 only).

## Limitations

- Cannot force protocol DUP on outgoing ACK publish
- Autonomous mode always ACKs immediately
- Controlled mode requires terminal interaction per command
