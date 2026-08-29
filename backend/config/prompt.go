package config

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func PromptInteractive(r *bufio.Reader) (*Config, error) {
	cfg := &Config{
		MQTTVersion: "5",
		ClientID:    DefaultClientID,
	}

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("HeyEV Backend — Configuration")
	fmt.Println("========================================")
	fmt.Println("Answer y/n for each option. Press Enter for the default shown in [brackets].")
	fmt.Println()

	qos1, err := promptYesNo(r, "Use QoS 1 instead of QoS 0?", false)
	if err != nil {
		return nil, err
	}
	if qos1 {
		cfg.QoS = 1
	}

	cfg.Retain, err = promptYesNo(r, "Set RETAIN=true on command publish?", false)
	if err != nil {
		return nil, err
	}

	cfg.Dup, err = promptYesNo(r, "Enable DUP flag experiment logging?", false)
	if err != nil {
		return nil, err
	}

	cfg.AllowDuplicatePublish, err = promptYesNo(r, "Allow duplicate command publish (skip idempotency block)?", false)
	if err != nil {
		return nil, err
	}

	cfg.AutoReconnect, err = promptYesNo(r, "Enable auto-reconnect on connection loss?", true)
	if err != nil {
		return nil, err
	}

	cfg.Debug, err = promptYesNo(r, "Enable debug logging?", false)
	if err != nil {
		return nil, err
	}

	fmt.Println()
	fmt.Println("MQTT Version: 5 (only version supported by this client)")
	fmt.Println("========================================")
	fmt.Println()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if err := cfg.loadEnvironment(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func promptYesNo(r *bufio.Reader, question string, defaultYes bool) (bool, error) {
	defaultLabel := "n"
	if defaultYes {
		defaultLabel = "y"
	}

	for {
		fmt.Printf("%s [%s]: ", question, defaultLabel)
		answer, err := readLine(r)
		if err != nil {
			return false, err
		}

		if answer == "" {
			return defaultYes, nil
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

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}
