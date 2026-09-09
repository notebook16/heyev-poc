# MQTT Command-Firing POC

**HeyEV Backend ↔ IoT Simulator**

**Selected approach: Option B** — MQTT 5, QoS 1, persistent session, session expiry, broker session queue.

This document updates the original QoS 0 / MQTT 3.1.1 POC. Both **Option A** (Retain + Message Expiry) and **Option B** (Session queue + Message Expiry) were implemented and tested on AWS IoT Core. **Option B was chosen** for the HeyEV command-firing architecture. **Option C** (AWS IoT Device Management Jobs / Commands) was evaluated on paper only and **not POC’d** because of high cost.

---

## 1. POC Objective

The objective of this POC is to validate the MQTT-based command-firing flow for the HeyEV IoT architecture using AWS IoT Core.

The POC validates communication between the HeyEV Backend and an IoT device through MQTT:

```
HeyEV Backend → AWS IoT Core → IoT Device → AWS IoT Core → HeyEV Backend
```

The POC specifically validates:

- HeyEV Backend can connect to AWS IoT Core with mutual TLS.
- IoT Simulator can connect to AWS IoT Core with mutual TLS.
- HeyEV Backend can publish a command using **MQTT 5** and **QoS 1**.
- IoT Simulator can receive the command (online and after offline reconnect).
- IoT Simulator can publish a device-level acknowledgement with `request_id`.
- HeyEV Backend can receive the acknowledgement and correlate it to the command.
- Offline devices can receive queued commands via a **persistent MQTT session** and **session expiry** (Option B).
- Last-command delivery via **topic retain + message expiry** works on an exact topic subscribe (Option A — tested, not selected).

MQTT 3.1.1 is **not** used. The client library (`github.com/eclipse/paho.golang`) is **MQTT 5 only**.

---

## 2. POC Architecture

The IoT Simulator represents the actual IoT device for this POC.

```
 ┌──────────────────┐         TLS :8883          ┌─────────────────┐
 │  HeyEV Backend   │◄──────────────────────────►│                 │
 │  (Go, MQTT 5)    │  publish commands QoS 1    │  AWS IoT Core   │
 │  heyev-backend-  │  subscribe ACKs            │  ap-south-1     │
 │  poc             │                            │                 │
 └──────────────────┘                            │  Things:        │
                                                 │  - heyev-       │
 ┌──────────────────┐         TLS :8883          │    backend-poc  │
 │  IoT Simulator   │◄──────────────────────────►│  - iot-         │
 │  (Go, MQTT 5)    │  subscribe commands QoS 1  │    simulator-poc│
 │  iot-simulator-  │  publish ACKs              │                 │
 │  poc             │                            └─────────────────┘
 └──────────────────┘
```

---

## 3. MQTT Configuration (selected: Option B)

| Parameter | Value |
|---|---|
| Cloud broker | AWS IoT Core (`ap-south-1`) |
| Protocol | **MQTT 5** |
| Transport | MQTT over TLS (port 8883), mutual authentication |
| Command delivery mode | **Option B — Session queue + Message Expiry** |
| Command QoS | **1** |
| ACK QoS | **1** |
| RETAIN | **false** (forced for Option B) |
| Persistent session (device) | **true** (Clean Start false on resume) |
| Session expiry (device) | **900 seconds (15 minutes)** default; must be > 0 |
| Message expiry | Optional per command (MQTT 5 `MessageExpiryInterval`); `0` = no expiry |
| Payload | JSON / UTF-8 |
| Implementation | Go (`paho.golang` + `autopaho`) |

QoS 1 gives MQTT-layer PUBACK (transport confirmation). That is **not** a device ACK. The simulator still publishes an application-level ACK with the same `request_id`.

---

## 4. Delivery options evaluated

Three options were considered for “command arrives even if the device was offline.”

### Option A — Retain + Message Expiry (tested, not selected)

- Backend publishes the command with **RETAIN=true** and optional MQTT 5 message expiry.
- AWS IoT stores **one** last message per topic.
- A late subscriber receives that snapshot **only if it subscribes to the exact topic** (no `+` / `#`).
- AWS IoT policy must include `iot:RetainPublish` (in addition to `iot:Publish`). Missing it causes DISCONNECT reason **135 (Not authorized)**.
- **Result:** PASS for exact-topic late subscribe. **Not selected** because retain is a last-value snapshot (Ctrl+R / second publish overwrites the first), and wildcard command subscribe (`heyev/v1/devices/+/commands`) does **not** receive retained messages on AWS IoT Core.

### Option B — Persistent session + session queue + Message Expiry (**selected**)

- Simulator connects with **QoS 1 subscribe**, **persistent session**, **session expiry > 0**, **stable Client ID**.
- Simulator must subscribe **once** while online, then may disconnect.
- Backend publishes **QoS 1**, **retain=false**.
- AWS IoT queues matching QoS 1 messages on the device session until reconnect or session/message expiry.
- Wildcard subscribe **is** valid for the queue (unlike retain).
- On reconnect: `SessionPresent (CONNACK): true` and queued command delivery.
- **Result:** Selected for HeyEV command firing. Matches production “device was offline, then came back” behaviour without Device Management Jobs.

### Option C — AWS IoT Device Management (Jobs / Commands) (**not POC’d — high cost**)

AWS IoT Device Management **Jobs** and **Commands** provide cloud-side job documents, per-device execution state (`QUEUED`, `IN_PROGRESS`, `SUCCEEDED`, `TIMED_OUT`, `CANCELED`), retries, and fleet targeting.

They were **not implemented** in this POC because:

- Each remote action / command execution is billed **on top of** MQTT messaging.
- List price for Device Jobs is **$0.003 per remote action** for the first 250,000 actions/month (then $0.0015).
- 10,000 devices × 2 commands/day ≈ **600,000 executions/month ≈ $1,800/month in Jobs fees alone**, plus MQTT publish, PUBACK, and status messages.
- HeyEV already has application-level ACK and `request_id` correlation; Jobs would duplicate orchestration at a high unit cost.

See **§16 Cost comparison**.

---

## 5. AWS IoT Core Setup

Two AWS IoT Core Things were created:

- `heyev-backend-poc`
- `iot-simulator-poc`

Each Thing has its own X.509 certificate and IoT policy. Certificates are used by the Go applications for mutual TLS to AWS IoT Core.

### HeyEV Backend policy (role)

- `iot:Connect`
- `iot:Publish` on `heyev/v1/devices/*/commands`
- `iot:RetainPublish` on command topics **only if Option A is used**
- `iot:Subscribe` / `iot:Receive` on `heyev/v1/devices/*/ack`

### IoT Simulator policy (role)

- `iot:Connect`
- `iot:Subscribe` / `iot:Receive` on `heyev/v1/devices/*/commands`
- `iot:Publish` on `heyev/v1/devices/*/ack`
- `iot:Publish` on `heyev/v1/devices/*/telemetry` (optional telemetry publish from the simulator)
- `iot:RetainPublish` on ACK topics **only if ACK retain is tested** (not required for Option B)

`iot:GetRetainedMessage` is **not** required for MQTT subscribe. It is an AWS API/console lookup of a stored retained message.

---

## 6. MQTT Topic Structure

**Command topic**

```
heyev/v1/devices/{device_id}/commands
```

Example: `heyev/v1/devices/6264/commands`

**Device ACK topic**

```
heyev/v1/devices/{device_id}/ack
```

Example: `heyev/v1/devices/6264/ack`

**Device telemetry topic** (simulator-only optional publish)

```
heyev/v1/devices/{device_id}/telemetry
```

Example (hardcoded POC sample): `heyev/v1/devices/866224084563153/telemetry`

`{device_id}` identifies the target device.

| Mode | Simulator subscribe | ACK publish |
|---|---|---|
| Autonomous | `heyev/v1/devices/+/commands` | `heyev/v1/devices/{id_from_topic}/ack` |
| Controlled | exact `.../{device_id}/commands` (or `+`) | same device id (or user-chosen ACK topic) |

**Option A retain:** subscribe must be the **exact** command topic (use controlled mode + concrete device ID).  
**Option B queue:** wildcard autonomous subscribe is valid.

---

## 7. Command Flow (Option B)

```
HeyEV Backend
      │
      │ PUBLISH QoS 1  retain=false
      │ heyev/v1/devices/6264/commands
      │ (+ optional Message Expiry Interval)
      ▼
 AWS IoT Core
      │
      │  if device online  → deliver now
      │  if device offline → queue on persistent session
      ▼
 IoT Simulator (same Client ID, QoS 1 subscribe)
```

Autonomous simulator subscribe:

```
heyev/v1/devices/+/commands
```

ACK is still sent to the **same** `device_id` parsed from the command topic. Controlled mode is only required if an operator must accept/skip ACK interactively.

---

## 8. Command Payload

JSON. Application correlation uses `request_id`.

```json
{
  "request_id": "cmd-a3e5f9d8-4b46-4a6c-b280-c1f9a161e169",
  "device_id": "6264",
  "command": "charging",
  "value": "on",
  "timestamp": "2026-08-31T11:08:06.81383378Z"
}
```

If Request ID is left blank, the backend generates `cmd-` + UUID.

---

## 9. Device ACK Flow

After receiving the command, the simulator publishes an application-level ACK to `heyev/v1/devices/{device_id}/ack`.

```json
{
  "request_id": "cmd-a3e5f9d8-4b46-4a6c-b280-c1f9a161e169",
  "device_id": "6264",
  "status": "ACKNOWLEDGED",
  "message": "Command received by simulator",
  "timestamp": "2026-08-31T11:08:06.830000000Z"
}
```

`request_id` is copied from the command so the backend can match ACK → command.

```
HeyEV Backend
      │
      │ Command (QoS 1)
      ▼
AWS IoT Core
      │
      │ Command (live or session queue)
      ▼
IoT Simulator
      │
      │ ACKNOWLEDGED (QoS 1) + same request_id
      ▼
AWS IoT Core
      │
      │ ACK
      ▼
HeyEV Backend
```

Delivery stages observed in the POC:

1. MQTT PUBLISH completed (socket send).
2. MQTT PUBACK (QoS 1 transport) — **not** device receipt.
3. Device-level ACK on the ack topic — simulator received the command.
4. Optional `SUCCESS` — only if the device reports success (simulator uses `ACKNOWLEDGED`).

---

## 10. POC Implementation

Both sides are Go applications using Eclipse Paho MQTT 5 (`paho.golang` / `autopaho`).

### HeyEV Backend

- Loads Amazon Root CA, device certificate, and private key.
- TLS mutual auth to AWS IoT Core.
- Interactive config: Option A or B, QoS, retain (A only), session, debug.
- Subscribes to ACK topic for the chosen device ID (or `+`).
- Publishes commands with QoS 1 (Option B), retain false, optional message expiry.
- Idempotency on `request_id` (duplicate command publish can be allowed for experiments).
- Ctrl+R repeats the last command with a **new** `request_id`.

### IoT Simulator

- Loads its own certificate and private key.
- TLS mutual auth to AWS IoT Core.
- Autonomous (auto ACK) or controlled (operator ACK).
- Option B: QoS 1 subscribe, persistent session, session expiry (default 900s), stable Client ID `iot-simulator-poc`.
- Parses `device_id` from the command topic and ACKs that same id.
- Optional hardcoded telemetry publish to `heyev/v1/devices/866224084563153/telemetry` after connect (startup y/n).
- Ctrl+R repeats the **last publish** (telemetry or ACK), same payload/QoS/retain settings.

---

## 11. POC Tests

### 11.1 Live command / ACK (baseline)

Device ID example: `6264` (also exercised with other IDs such as `12345` / `123456789`).

Backend publishes to `heyev/v1/devices/{id}/commands` with JSON including `request_id`.  
Simulator receives the command and publishes ACK to `heyev/v1/devices/{id}/ack`.  
Backend receives ACK and correlates `request_id`.

**Result: PASS**

### 11.2 Option A — Retain + Message Expiry (evaluated)

| Step | Result |
|---|---|
| Publish retain=true without `iot:RetainPublish` | FAIL — DISCONNECT **135 Not authorized**; QoS 0 still printed PUBLISH OK |
| Add `iot:RetainPublish` on backend command topic | PASS — connection stays up |
| Late subscribe on wildcard `heyev/v1/devices/+/commands` | FAIL — AWS IoT does **not** deliver retained messages to wildcard filters |
| Late subscribe on exact `heyev/v1/devices/{id}/commands` (controlled mode) | PASS — last retained command delivered |
| Second retain publish (Ctrl+R) | Replaces the first retained message (one snapshot per topic) |

**Decision:** Option A is valid for “last command on this exact topic” but not for a queue of missed commands. **Not selected.**

### 11.3 Option B — Persistent session + session queue (**selected**)

| Step | Result |
|---|---|
| Simulator Option B, QoS 1, persistent session, session expiry 900s, subscribe while online | Session created (`SessionPresent: false` on first connect is normal) |
| Simulator disconnect (Ctrl+C) | Session kept for session expiry (default 15 minutes) |
| Backend Option B, QoS 1, retain=false, publish while device offline | Command queued on the simulator session |
| Simulator reconnect, **same Client ID** | `SessionPresent: true`; queued command delivered; application ACK sent |
| Two simulators, same Client ID | FAIL — DISCONNECT **142 Session taken over** / CONNACK **151 Duplicate ClientId** |
| Session expiry elapsed (> 900s) before reconnect | Session gone; `SessionPresent: false`; queue discarded |
| Per-command message expiry (e.g. 120s) | Command dropped from the queue if not consumed in time, even if session still exists |

**Decision: Option B selected** for HeyEV command firing.

---

## 12. Current POC Status

| Test | Status |
|---|---|
| AWS IoT Core setup (two Things, certs, policies) | PASS |
| Backend certificate authentication | PASS |
| Simulator certificate authentication | PASS |
| Backend MQTT connection | PASS |
| Simulator MQTT connection | PASS |
| MQTT 5 | PASS |
| TLS (mTLS :8883) | PASS |
| Live command + application ACK + `request_id` | PASS |
| Option A retain + message expiry (exact topic) | PASS (evaluated) |
| Option A wildcard late subscribe | FAIL (AWS limitation) — expected |
| Option B QoS 1 | PASS |
| Option B persistent session | PASS |
| Option B session expiry | PASS (900s default) |
| Option B offline session queue | PASS |
| Option B selected as target architecture | **YES** |
| Option C Device Management Jobs / Commands | **Not POC’d (cost)** |

---

## 13. Option C — AWS IoT Device Management (not performed)

Option C would use **AWS IoT Jobs** and/or **Commands** instead of (or on top of) raw MQTT publish:

- Cloud creates a job/command document and targets things or groups.
- Each device execution is tracked (`QUEUED` → `IN_PROGRESS` → `SUCCEEDED` / `TIMED_OUT` / `CANCELED`).
- Timeouts, retries, and fleet rollouts are built in.
- MQTT is still used to deliver the job document and status; **Jobs/Commands fees apply in addition to messaging.**

**Why this POC did not implement it**

- Unit cost per remote action ($0.003 list, first 250k/month) is large at charger-fleet command volume (see §16).
- Application ACK + `request_id` already cover correlation for HeyEV command firing.
- Option B already covers offline delivery via the MQTT session queue.

Device Management remains a possible later fit for **fleet OTA / factory reset / bulk maintenance**, not for per-transaction charger commands.

---

## 14. Operational notes learned in the POC

- **One Client ID, one connection.** Duplicate `iot-simulator-poc` processes cause 142/151 reconnect loops. `grep` in `ps` is not a leftover process.
- **QoS 0 PUBLISH OK is not broker acceptance.** Retain without `iot:RetainPublish` still printed success, then DISCONNECT 135.
- **ACK retain** needs `iot:RetainPublish` on the **simulator** ACK topic; not needed for Option B.
- **Do not use `+` as device ID** for Option A retain tests.
- Simulator **autonomous ACK already uses the same `device_id`** as the command topic. Controlled mode is optional.

---

## 15. Conclusion

The POC validates MQTT command firing on AWS IoT Core using **MQTT 5**, **QoS 1**, and an application-level ACK with **`request_id`**.

**Option A** (retain + message expiry) was tested and works for last-command delivery on an **exact** topic subscribe. It was **not selected**.

**Option B** (persistent session + session expiry + QoS 1 session queue + optional message expiry) was tested and **selected**. Offline devices receive queued commands on reconnect with the same Client ID.

**Option C** (AWS IoT Device Management Jobs/Commands) was **not POC’d** due to high per-execution cost.

Selected flow:

```
HeyEV Backend
      │
      │ MQTT 5 PUBLISH / QoS 1 / retain=false
      ▼
AWS IoT Core  (live deliver, or queue on persistent device session)
      │
      ▼
IoT Device / Simulator  (QoS 1 subscribe, persistent session, session expiry)
      │
      │ Application ACK / QoS 1 / same request_id
      ▼
AWS IoT Core
      │
      ▼
HeyEV Backend
```

---

## 16. Cost comparison (Option A vs B vs C)

Prices below are **public AWS list rates** (US East / common IoT Core list: messaging **$1.00 per million messages** for the first 1 billion; connectivity **$0.08 per million connection-minutes**; Device Jobs **$0.003 per remote action** for the first 250,000/month, then $0.0015).  
This POC used **ap-south-1**; confirm current regional rates in the [AWS IoT Core](https://aws.amazon.com/iot-core/pricing/) and [Device Management](https://aws.amazon.com/iot-device-management/pricing/) pricing pages before budgeting.

MQTT messages are metered in **5 KB** increments. Keep-alive PING is not billed. QoS 1 PUBACK **is** billed as a message.

### Illustrative fleet: 10,000 devices, 2 commands/day, 1 ACK each, payloads &lt; 5 KB, devices connected 24/7

Monthly command volume: `10,000 × 2 × 30 = 600,000` commands (and 600,000 ACKs).

| Cost component | Option A (MQTT retain) | Option B (MQTT session queue) **selected** | Option C (Device Management Jobs/Commands) |
|---|---|---|---|
| How offline delivery is billed | MQTT messaging only (retain store is not a Jobs fee) | MQTT messaging only (queue is part of IoT Core session) | **$0.003 × each job/command execution** **plus** MQTT |
| Command + ACK MQTT (publish + deliver) | ~2.4M msgs | ~2.4M msgs | ~2.4M+ msgs (job document, status, completion) |
| Extra QoS 1 PUBACKs | If QoS 1 used | ~1.2M msgs | Yes (Commands docs include PUBACK) |
| Messaging @ $1 / million | **~$2–4 / month** | **~$3–5 / month** | **~$4–8 / month** (more status MQTT) |
| Connectivity 10k × 43,200 min @ $0.08 / million min | **~$35 / month** | **~$35 / month** | **~$35 / month** |
| Device Jobs / Commands executions | $0 | $0 | **600,000 × $0.003 ≈ $1,800 / month** (first 250k @ $0.003 + remainder @ $0.0015 ≈ **$1,275–$1,800**) |
| Fleet Indexing / bulk registration | Not required | Not required | Optional extra if used |
| **Indicative total (command path)** | **~$40 / month** | **~$40 / month** | **~$1,300–$1,850+ / month** |

Order of magnitude: **Option C is ~30–45× more expensive** than MQTT Option A/B for the same command volume, because Jobs/Commands charge **per execution**, not just per MQTT message.

### Qualitative cost / capability

| | Option A | Option B (chosen) | Option C (not POC’d) |
|---|---|---|---|
| IoT Core MQTT | Yes | Yes | Yes (still required) |
| Extra AWS product fee | No | No | **Yes — Jobs/Commands** |
| Offline behaviour | Last retained command only | Queue of QoS 1 messages for the session | Job stays QUEUED until device runs it (longer, richer state) |
| Wildcard subscribe | Retained snapshot **not** delivered | Queued messages **are** delivered | N/A (Jobs topics / reserved topics) |
| App ACK + `request_id` | Built in POC | Built in POC | Jobs already track execution state (overlap) |
| Good fit | Last config / last command snapshot | **Per-command fire-and-ack, including brief offline** | Fleet OTA, factory reset, bulk maintenance |
| POC | Done | **Done — selected** | Skipped (cost) |

Option A vs Option B **AWS bill is essentially the same** (both are IoT Core connectivity + messaging). Option B was chosen for **correctness** (session queue vs last-value retain), not for a lower MQTT price.

Option C was rejected for **command firing** on cost: hundreds of thousands of charger commands per month would incur a large **per-remote-action** bill on top of the MQTT traffic HeyEV must pay anyway.
