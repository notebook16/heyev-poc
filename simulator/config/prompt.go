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
	fmt.Println("IoT Simulator — Configuration")
	fmt.Println("========================================")
	fmt.Println("Answer y/n or pick a numbered option. Press Enter for the default in [brackets].")
	fmt.Println()

	mode, err := promptMode(r)
	if err != nil {
		return nil, err
	}
	cfg.Mode = mode

	qos1, err := promptYesNo(r, "Use QoS 1 instead of QoS 0 for ACK publish?", false)
	if err != nil {
		return nil, err
	}
	if qos1 {
		cfg.QoS = 1
	}

	cfg.Retain, err = promptYesNo(r, "Set RETAIN=true on ACK publish?", false)
	if err != nil {
		return nil, err
	}

	cfg.Dup, err = promptYesNo(r, "Log incoming MQTT DUP flag on commands?", false)
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

func promptMode(r *bufio.Reader) (Mode, error) {
	fmt.Println("Simulator mode:")
	fmt.Println("  1) Autonomous — auto ACK on every command")
	fmt.Println("  2) Controlled  — you choose when/how to ACK")

	for {
		fmt.Print("Select mode [1]: ")
		answer, err := readLine(r)
		if err != nil {
			return "", err
		}
		if answer == "" {
			answer = "1"
		}

		switch answer {
		case "1", "autonomous":
			return ModeAutonomous, nil
		case "2", "controlled":
			return ModeControlled, nil
		default:
			fmt.Println("Please enter 1 or 2.")
		}
	}
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
