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

	log.Config("Backend is listening for ACKs. Enter commands below (Ctrl+C to stop).")

	for {
		commandDeviceID := cfg.CommandDeviceID
		if cfg.WildcardSubscribe {
			commandDeviceID, err = promptLine(reader, "Command device ID:")
			if err != nil {
				log.Error("%v", err)
				continue
			}
			if err := cfg.SetCommandDeviceID(commandDeviceID); err != nil {
				log.Error("%v", err)
				continue
			}
		}

		commandName, err := promptLine(reader, "Command:")
		if err != nil {
			log.Error("%v", err)
			continue
		}
		if strings.TrimSpace(commandName) == "" {
			continue
		}

		value, err := promptLine(reader, "Value:")
		if err != nil {
			log.Error("%v", err)
			continue
		}

		expiryStr, err := promptLine(reader, "Message expiry in seconds (0 = no expiry):")
		if err != nil {
			log.Error("%v", err)
			continue
		}
		expirySec, err := strconv.ParseUint(strings.TrimSpace(expiryStr), 10, 32)
		if err != nil {
			log.Error("Invalid expiry value: %v", err)
			continue
		}

		requestID, err := promptLine(reader, "Request ID (blank = generate):")
		if err != nil {
			log.Error("%v", err)
			continue
		}

		cmd, err := cmdService.Build(commandDeviceID, commandName, value, strings.TrimSpace(requestID))
		if err != nil {
			log.Error("%v", err)
			continue
		}

		if !cmdService.CanPublish(cmd.RequestID) {
			continue
		}

		tracker.Set(cmd.RequestID, state.StatusPending)

		payload, err := cmdService.Marshal(cmd)
		if err != nil {
			log.Error("%v", err)
			continue
		}

		publishTopic := cfg.CommandPublishTopic
		if publishTopic == "" {
			publishTopic = cfg.CommandTopicFor(commandDeviceID)
		}

		log.Publish("Request ID: %s", cmd.RequestID)
		log.Config("MQTT Version: 5")

		pubCtx, pubCancel := context.WithTimeout(context.Background(), 30*time.Second)
		result := mqttclient.PublishCommand(pubCtx, cfg, log, mqttClient.Manager(), publishTopic, payload, uint32(expirySec))
		pubCancel()

		if result.Err == nil {
			cmdService.MarkPublished(cmd.RequestID)
			log.Publish("Waiting for device-level ACK on %s ...", cfg.AckSubscribeTopic)
		}
	}
}

func printStartupConfig(log *logger.Logger, cfg *config.Config) {
	log.Config("Starting HeyEV Backend POC")
	log.Config("MQTT Version: 5")
	log.Config("QoS: %d", cfg.QoS)
	log.Config("Retain: %t", cfg.Retain)
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
