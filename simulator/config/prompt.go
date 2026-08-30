package config

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
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

	mode, err := promptSimulatorMode(r)
	if err != nil {
		return nil, err
	}
	cfg.Mode = mode

	delivery, err := promptDeliveryMode(r)
	if err != nil {
		return nil, err
	}
	cfg.DeliveryMode = delivery

	clientID, err := promptLineOptional(r, "Client ID (blank = default)", DefaultClientID)
	if err != nil {
		return nil, err
	}
	cfg.ClientID = clientID

	qosDefault := delivery == DeliveryModeB
	qos1, err := promptYesNo(r, "Use QoS 1 for command subscription and ACK publish? (required for Option B)", qosDefault)
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

	sessionDefault := uint32(0)
	persistentDefault := false
	if delivery == DeliveryModeB {
		sessionDefault = 900
		persistentDefault = true
	}

	cfg.SessionExpirySec, err = promptUint(r, "Session expiry in seconds (0 = session ends on disconnect)", sessionDefault)
	if err != nil {
		return nil, err
	}

	cfg.PersistentSession, err = promptYesNo(r, "Use persistent MQTT session (offline command queue)?", persistentDefault)
	if err != nil {
		return nil, err
	}

	cfg.Debug, err = promptYesNo(r, "Enable debug logging?", false)
	if err != nil {
		return nil, err
	}

	fmt.Println()
	fmt.Println("MQTT Version: 5 (only version supported by this client)")
	fmt.Println("Message Expiry on commands is set by the backend when it publishes.")
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

func promptSimulatorMode(r *bufio.Reader) (Mode, error) {
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

func promptDeliveryMode(r *bufio.Reader) (DeliveryMode, error) {
	fmt.Println("Command delivery mode (must match backend):")
	fmt.Println("  1) Option A — Retain + Message Expiry")
	fmt.Println("  2) Option B — Session queue + Message Expiry")

	for {
		fmt.Print("Select delivery mode [2]: ")
		answer, err := readLine(r)
		if err != nil {
			return "", err
		}
		if answer == "" {
			answer = "2"
		}

		switch answer {
		case "1", "a", "A":
			return DeliveryModeA, nil
		case "2", "b", "B":
			return DeliveryModeB, nil
		default:
			fmt.Println("Please enter 1 or 2.")
		}
	}
}

func promptLineOptional(r *bufio.Reader, label, defaultVal string) (string, error) {
	fmt.Printf("%s [%s]: ", label, defaultVal)
	answer, err := readLine(r)
	if err != nil {
		return "", err
	}
	if answer == "" {
		return defaultVal, nil
	}
	return answer, nil
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

func promptUint(r *bufio.Reader, question string, defaultVal uint32) (uint32, error) {
	for {
		fmt.Printf("%s [%d]: ", question, defaultVal)
		answer, err := readLine(r)
		if err != nil {
			return 0, err
		}
		if answer == "" {
			return defaultVal, nil
		}
		v, err := strconv.ParseUint(strings.TrimSpace(answer), 10, 32)
		if err != nil {
			fmt.Println("Please enter a non-negative integer.")
			continue
		}
		return uint32(v), nil
	}
}

func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}
