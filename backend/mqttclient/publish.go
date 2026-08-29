package mqttclient

import (
	"context"
	"fmt"
	"time"

	"github.com/eclipse/paho.golang/paho"

	"heyev-backend-poc/config"
	"heyev-backend-poc/logger"
)

type PublishResult struct {
	Topic     string
	RequestID string
	QoS       byte
	Retain    bool
	ExpirySec uint32
	Payload   []byte
	Timestamp time.Time
	Reason    byte
	Err       error
}

func PublishCommand(ctx context.Context, cfg *config.Config, log *logger.Logger, cm interface {
	Publish(context.Context, *paho.Publish) (*paho.PublishResponse, error)
}, topic string, payload []byte, expirySec uint32) PublishResult {
	result := PublishResult{
		Topic:     topic,
		QoS:       cfg.QoS,
		Retain:    cfg.Retain,
		ExpirySec: expirySec,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}

	pub := &paho.Publish{
		Topic:   topic,
		QoS:     cfg.QoS,
		Retain:  cfg.Retain,
		Payload: payload,
	}

	if cfg.Dup {
		log.Publish("DUP flag requested but cannot be set on outgoing PUBLISH with paho.golang (protocol DUP is broker/session managed)")
	}

	if expirySec > 0 {
		pub.Properties = &paho.PublishProperties{
			MessageExpiry: &expirySec,
		}
		log.Expiry("Message Expiry Interval: %ds (MQTT 5 broker-level)", expirySec)
	} else {
		log.Expiry("Message Expiry: none (0 = no expiry)")
	}

	log.Publish("Publishing command...")
	log.Publish("Topic: %s", topic)
	log.Publish("QoS: %d", cfg.QoS)
	log.Publish("Retain: %t", cfg.Retain)
	if cfg.Retain {
		log.Publish("NOTE: RETAIN=true — this is a retained MQTT message, not a normal transient publish")
	}
	log.Publish("Payload: %s", string(payload))
	log.Publish("Timestamp: %s", result.Timestamp.Format(time.RFC3339Nano))

	resp, err := cm.Publish(ctx, pub)
	result.Err = err
	if err != nil {
		log.Error("MQTT PUBLISH failed: %v", err)
		log.Publish("Publish result: FAILED")
		return result
	}

	log.Publish("Stage 1: MQTT PUBLISH completed successfully")
	log.Publish("Publish result: OK")

	if cfg.QoS >= 1 && resp != nil {
		result.Reason = resp.ReasonCode
		log.PubAck("Stage 2: MQTT transport PUBACK received (reason=%d) — this is NOT a device-level ACK", resp.ReasonCode)
	} else {
		log.Publish("Stage 2: No MQTT PUBACK (QoS 0) — transport delivery is best-effort")
	}

	return result
}

func FormatPublishSummary(r PublishResult) string {
	if r.Err != nil {
		return fmt.Sprintf("failed: %v", r.Err)
	}
	return "success"
}
