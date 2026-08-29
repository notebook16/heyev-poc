package command

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/paho"

	"iot-simulator-poc/ack"
	"iot-simulator-poc/config"
	"iot-simulator-poc/logger"
	"iot-simulator-poc/topic"
)

type Publisher interface {
	PublishACK(ctx context.Context, topic string, payload []byte) error
}

type Handler struct {
	cfg     *config.Config
	log     *logger.Logger
	pub     Publisher
	queue   chan *IncomingCommand
	seenIDs map[string]struct{}
	seenMu  sync.Mutex
	reader  *bufio.Reader
}

func NewHandler(cfg *config.Config, log *logger.Logger, pub Publisher, reader *bufio.Reader) *Handler {
	return &Handler{
		cfg:     cfg,
		log:     log,
		pub:     pub,
		queue:   make(chan *IncomingCommand, 64),
		seenIDs: make(map[string]struct{}),
		reader:  reader,
	}
}

func (h *Handler) OnMessage(msg *paho.Publish) {
	cmd, err := Parse(msg)
	if err != nil {
		h.log.Error("Failed to parse command: %v", err)
		return
	}

	h.seenMu.Lock()
	_, dup := h.seenIDs[cmd.RequestID]
	h.seenMu.Unlock()
	cmd.DuplicateRequest = dup

	if h.cfg.Mode == config.ModeAutonomous {
		h.handleAutonomous(cmd)
		return
	}

	select {
	case h.queue <- cmd:
	default:
		h.log.Error("Command queue full, dropping message request_id=%s", cmd.RequestID)
	}
}

func (h *Handler) RunControlledLoop(ctx context.Context) {
	h.log.Simulator("Controlled mode: waiting for commands...")

	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-h.queue:
			h.presentControlledCommand(ctx, cmd)
		}
	}
}

func (h *Handler) handleAutonomous(cmd *IncomingCommand) {
	h.logCommand(cmd)

	payload, err := ack.Build(cmd.DeviceID, cmd.RequestID)
	if err != nil {
		h.log.Error("Build ACK failed: %v", err)
		return
	}

	ackTopic := topic.AckTopic(cmd.DeviceID)
	h.log.Ack("Request ID: %s", cmd.RequestID)
	h.log.Ack("Status: ACKNOWLEDGED")

	pubCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := h.pub.PublishACK(pubCtx, ackTopic, payload); err != nil {
		return
	}

	h.markSeen(cmd.RequestID)
}

func (h *Handler) presentControlledCommand(ctx context.Context, cmd *IncomingCommand) {
	h.log.Command("========================================")
	h.log.Command("COMMAND RECEIVED")
	h.log.Command("========================================")
	h.log.Command("Topic: %s", cmd.Topic)
	h.log.Command("Device ID: %s", cmd.DeviceID)
	h.log.Command("Request ID: %s", cmd.RequestID)
	h.log.Command("Duplicate request_id: %s", yesNo(cmd.DuplicateRequest))
	h.log.Command("MQTT DUP flag: %t", cmd.MQTTDuplicate)
	h.log.Command("QoS: %d", cmd.QoS)
	h.log.Command("Retain: %t", cmd.Retain)
	h.log.Command("Payload: %s", cmd.Payload)
	h.log.Command("========================================")

	defaultTopic := topic.AckTopic(cmd.DeviceID)

	accept, err := promptYesNo(h.reader, "Accept this command? (y/n):")
	if err != nil {
		h.log.Error("%v", err)
		return
	}
	if !accept {
		h.log.Command("Command ignored by user")
		return
	}

	sendACK, err := promptYesNo(h.reader, "Send ACK? (y/n):")
	if err != nil {
		h.log.Error("%v", err)
		return
	}
	if !sendACK {
		h.log.Command("ACK skipped by user")
		h.markSeen(cmd.RequestID)
		return
	}

	ackTopic, err := promptLine(h.reader, fmt.Sprintf("ACK topic [%s]:", defaultTopic))
	if err != nil {
		h.log.Error("%v", err)
		return
	}
	if strings.TrimSpace(ackTopic) == "" {
		ackTopic = defaultTopic
	}

	payload, err := ack.Build(cmd.DeviceID, cmd.RequestID)
	if err != nil {
		h.log.Error("Build ACK failed: %v", err)
		return
	}

	h.log.Ack("Request ID: %s", cmd.RequestID)
	h.log.Ack("Status: ACKNOWLEDGED")

	pubCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := h.pub.PublishACK(pubCtx, ackTopic, payload); err != nil {
		return
	}

	h.markSeen(cmd.RequestID)
}

func (h *Handler) logCommand(cmd *IncomingCommand) {
	h.log.Command("Received")
	h.log.Command("Topic: %s", cmd.Topic)
	h.log.Command("Device ID: %s", cmd.DeviceID)
	h.log.Command("Request ID: %s", cmd.RequestID)
	h.log.Command("QoS: %d", cmd.QoS)
	h.log.Command("Retain: %t", cmd.Retain)
	if h.cfg.Dup {
		h.log.Command("MQTT DUP flag: %t", cmd.MQTTDuplicate)
	}
	h.log.Command("Payload: %s", cmd.Payload)
}

func (h *Handler) markSeen(requestID string) {
	h.seenMu.Lock()
	defer h.seenMu.Unlock()
	h.seenIDs[requestID] = struct{}{}
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

func promptLine(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s\n> ", label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func promptYesNo(reader *bufio.Reader, label string) (bool, error) {
	for {
		answer, err := promptLine(reader, label)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(answer) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("Please answer y or n.")
		}
	}
}
