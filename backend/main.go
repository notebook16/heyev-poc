package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"heyev-backend-poc/ack"
	"heyev-backend-poc/command"
	"heyev-backend-poc/config"
	"heyev-backend-poc/logger"
	"heyev-backend-poc/mqttclient"
	"heyev-backend-poc/state"
	"heyev-backend-poc/tlsconfig"
)

func main() {
	log := logger.New()
	reader := bufio.NewReader(os.Stdin)

	cfg, err := config.PromptInteractive(reader)
	if err != nil {
		log.Error("%v", err)
		os.Exit(1)
	}

	printStartupConfig(log, cfg)
	deviceID, err := promptLine(reader, "Device ID (use + for all devices):")
	if err != nil {
		log.Error("%v", err)
		os.Exit(1)
	}

	if err := cfg.SetDeviceTopics(deviceID); err != nil {
		log.Error("%v", err)
		os.Exit(1)
	}

	log.Config("ACK subscribe topic: %s", cfg.AckSubscribeTopic)
	if cfg.WildcardSubscribe {
		log.Config("Wildcard subscribe enabled — command publish requires a concrete device ID per command")
	} else {
		log.Config("Command publish topic: %s", cfg.CommandPublishTopic)
	}

	tlsCfg, err := tlsconfig.Load(cfg.Endpoint)
	if err != nil {
		log.TLS("%v", err)
		os.Exit(1)
	}
	log.TLS("TLS configured with mutual authentication")

	tracker := state.NewTracker()
	cmdStore := command.NewIdempotencyStore()
	ackStore := ack.NewIdempotencyStore()
	ackHandler := ack.NewHandler(log, ackStore, tracker)
	cmdService := command.NewService(log, cmdStore, tracker, cfg.AllowDuplicatePublish)

	mqttClient := mqttclient.New(cfg, log, ackHandler.Handle)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mqttClient.Connect(ctx, tlsCfg, cfg.AckSubscribeTopic); err != nil {
		log.Connect("%v", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Disconnect("Stopping HeyEV Backend POC...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = mqttClient.Disconnect(shutdownCtx)
		log.Disconnect("Disconnected from AWS IoT Core.")
		os.Exit(0)
	}()

	log.Config("ACK subscribe ready on %s", cfg.AckSubscribeTopic)
	if cfg.DeliveryMode == config.DeliveryModeA && !cfg.EffectiveRetain() {
		log.Config("NOTE: retain=false — the simulator must already be subscribed. For late-subscribe Option A, enable RETAIN=true.")
	}
	log.Config("Device ID is set. Now publish a command: name, value, expiry seconds, optional request ID.")
	log.Config("After each command, press Ctrl+R (or r) to send it again. Enter for a new command. Ctrl+C to stop.")

	var last *savedCommand

	for {
		if last != nil {
			repeat, err := waitRepeatOrNew(reader, *last)
			if err != nil {
				log.Error("%v", err)
				continue
			}
			if repeat {
				log.Config("Repeating last command with a new request_id")
				publishSaved(cfg, log, cmdService, tracker, mqttClient, *last, "")
				continue
			}
		}

		if err := promptAndPublish(reader, cfg, log, cmdService, tracker, mqttClient, &last, ""); err != nil {
			log.Error("%v", err)
		}
	}
}

type savedCommand struct {
	DeviceID  string
	Name      string
	Value     string
	ExpirySec uint32
}

func promptAndPublish(
	reader *bufio.Reader,
	cfg *config.Config,
	log *logger.Logger,
	cmdService *command.Service,
	tracker *state.Tracker,
	mqttClient *mqttclient.Client,
	last **savedCommand,
	commandName string,
) error {
	commandDeviceID := cfg.CommandDeviceID
	if cfg.WildcardSubscribe {
		id, err := promptLine(reader, "Command device ID:")
		if err != nil {
			return err
		}
		if err := cfg.SetCommandDeviceID(id); err != nil {
			return err
		}
		commandDeviceID = cfg.CommandDeviceID
	}

	commandName = strings.TrimSpace(commandName)
	if commandName == "" {
		var err error
		commandName, err = promptLine(reader, "Command name (example: lock):")
		if err != nil {
			return err
		}
	}
	if commandName == "" {
		return fmt.Errorf("command name cannot be empty. Example: lock")
	}
	if isRepeatInput(commandName) {
		if last == nil || *last == nil {
			return fmt.Errorf("no previous command to repeat yet")
		}
		log.Config("Repeating last command with a new request_id")
		publishSaved(cfg, log, cmdService, tracker, mqttClient, **last, "")
		return nil
	}

	value, err := promptLine(reader, "Value (example: on):")
	if err != nil {
		return err
	}

	expiryStr, err := promptLine(reader, "Message expiry in seconds (0 = no expiry):")
	if err != nil {
		return err
	}
	expirySec, err := strconv.ParseUint(strings.TrimSpace(expiryStr), 10, 32)
	if err != nil {
		return fmt.Errorf("invalid expiry value: %w", err)
	}

	requestID, err := promptLine(reader, "Request ID (blank = generate):")
	if err != nil {
		return err
	}

	saved := savedCommand{
		DeviceID:  commandDeviceID,
		Name:      commandName,
		Value:     value,
		ExpirySec: uint32(expirySec),
	}
	*last = &saved
	publishSaved(cfg, log, cmdService, tracker, mqttClient, saved, requestID)
	return nil
}

func publishSaved(
	cfg *config.Config,
	log *logger.Logger,
	cmdService *command.Service,
	tracker *state.Tracker,
	mqttClient *mqttclient.Client,
	saved savedCommand,
	requestID string,
) {
	if cfg.WildcardSubscribe {
		if err := cfg.SetCommandDeviceID(saved.DeviceID); err != nil {
			log.Error("%v", err)
			return
		}
	}

	cmd, err := cmdService.Build(saved.DeviceID, saved.Name, saved.Value, strings.TrimSpace(requestID))
	if err != nil {
		log.Error("%v", err)
		return
	}

	if !cmdService.CanPublish(cmd.RequestID) {
		return
	}

	tracker.Set(cmd.RequestID, state.StatusPending)

	payload, err := cmdService.Marshal(cmd)
	if err != nil {
		log.Error("%v", err)
		return
	}

	publishTopic := cfg.CommandPublishTopic
	if publishTopic == "" {
		publishTopic = cfg.CommandTopicFor(saved.DeviceID)
	}

	log.Publish("Request ID: %s", cmd.RequestID)
	log.Config("MQTT Version: 5")

	pubCtx, pubCancel := context.WithTimeout(context.Background(), 30*time.Second)
	result := mqttclient.PublishCommand(pubCtx, cfg, log, mqttClient.Manager(), publishTopic, payload, saved.ExpirySec)
	pubCancel()

	if result.Err == nil {
		cmdService.MarkPublished(cmd.RequestID)
		log.Publish("Waiting for device-level ACK on %s ...", cfg.AckSubscribeTopic)
	}
}

func waitRepeatOrNew(reader *bufio.Reader, last savedCommand) (bool, error) {
	fmt.Printf("Repeat last (%s / %s / %ds)? Press Ctrl+R (or r) to repeat, Enter for a new command:\n> ", last.Name, last.Value, last.ExpirySec)

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("read input: %w", err)
		}
		return isRepeatInput(strings.TrimSpace(line)), nil
	}

	old, err := term.MakeRaw(fd)
	if err != nil {
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, fmt.Errorf("read input: %w", err)
		}
		return isRepeatInput(strings.TrimSpace(line)), nil
	}
	defer func() {
		_ = term.Restore(fd, old)
	}()

	buf := make([]byte, 1)
	if _, err := os.Stdin.Read(buf); err != nil {
		return false, fmt.Errorf("read key: %w", err)
	}
	fmt.Print("\r\n")

	switch buf[0] {
	case 0x12, 'r', 'R':
		return true, nil
	case 0x03:
		_ = term.Restore(fd, old)
		p, findErr := os.FindProcess(os.Getpid())
		if findErr == nil {
			_ = p.Signal(os.Interrupt)
		}
		return false, fmt.Errorf("interrupted")
	case '\r', '\n':
		return false, nil
	default:
		return isRepeatInput(string(buf)), nil
	}
}

func isRepeatInput(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.ContainsRune(s, '\x12') {
		return true
	}
	switch strings.ToLower(s) {
	case "r", "repeat", "^r":
		return true
	default:
		return false
	}
}

func printStartupConfig(log *logger.Logger, cfg *config.Config) {
	log.Config("Starting HeyEV Backend POC")
	log.Config("Delivery mode: %s", cfg.DeliveryMode.Label())
	log.Config("MQTT Version: 5")
	log.Config("QoS: %d", cfg.QoS)
	log.Config("Retain: %t (effective on publish: %t)", cfg.Retain, cfg.EffectiveRetain())
	log.Config("Session expiry: %ds", cfg.SessionExpirySec)
	log.Config("Persistent session: %t", cfg.PersistentSession)
	log.Config("DUP: %t", cfg.Dup)
	if cfg.Dup {
		log.Config("DUP experiment: protocol DUP cannot be forced on outgoing publish with paho.golang")
	}
	log.Config("Message Expiry: supported (MQTT 5)")
	log.Config("Client ID: %s", cfg.ClientID)
	log.Config("Endpoint: %s", cfg.Endpoint)
	log.Config("Allow duplicate publish: %t", cfg.AllowDuplicatePublish)
	log.Config("Auto reconnect: %t", cfg.AutoReconnect)
}

func promptLine(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s\n> ", label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}
