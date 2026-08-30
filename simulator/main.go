package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"iot-simulator-poc/command"
	"iot-simulator-poc/config"
	"iot-simulator-poc/logger"
	"iot-simulator-poc/mqttclient"
	"iot-simulator-poc/tlsconfig"
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

	subscribeTopic := cfg.DefaultCommandTopic()

	if cfg.Mode == config.ModeControlled {
		deviceID, err := promptLine(reader, "Device ID (use + for all devices):")
		if err != nil {
			log.Error("%v", err)
			os.Exit(1)
		}
		if err := cfg.SetCommandSubscribeTopic(deviceID); err != nil {
			log.Error("%v", err)
			os.Exit(1)
		}
		subscribeTopic = cfg.CommandTopic
	}

	tlsCfg, err := tlsconfig.Load(cfg.Endpoint)
	if err != nil {
		log.TLS("%v", err)
		os.Exit(1)
	}
	log.TLS("TLS configured with mutual authentication")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mqttClient := mqttclient.New(cfg, log, nil)
	cmdHandler := command.NewHandler(cfg, log, mqttClient, reader)
	mqttClient.SetMessageHandler(cmdHandler.OnMessage)

	if err := mqttClient.Connect(ctx, tlsCfg, subscribeTopic); err != nil {
		log.Connect("%v", err)
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Disconnect("Stopping IoT Simulator...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = mqttClient.Disconnect(shutdownCtx)
		log.Disconnect("Disconnected from AWS IoT Core.")
		os.Exit(0)
	}()

	log.Simulator("Subscribed to %s", subscribeTopic)
	cmdHandler.Run(ctx)
}

func printStartupConfig(log *logger.Logger, cfg *config.Config) {
	log.Config("Starting IoT Simulator POC")
	log.Config("Mode: %s", cfg.Mode)
	log.Config("Delivery mode: %s", cfg.DeliveryMode.Label())
	log.Config("MQTT Version: 5")
	log.Config("QoS: %d", cfg.QoS)
	log.Config("Retain: %t", cfg.Retain)
	log.Config("Session expiry: %ds", cfg.SessionExpirySec)
	log.Config("Persistent session: %t", cfg.PersistentSession)
	log.Config("DUP: %t", cfg.Dup)
	log.Config("Message Expiry: N/A on simulator ACK publish (backend sets expiry on commands)")
	log.Config("Client ID: %s", cfg.ClientID)
	log.Config("Endpoint: %s", cfg.Endpoint)
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
