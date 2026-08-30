package command

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type lastACK struct {
	Topic     string
	Payload   []byte
	RequestID string
}

func waitRepeatOrNew(reader *bufio.Reader, last lastACK) (bool, error) {
	fmt.Printf("Repeat last ACK (request_id=%s)? Press Ctrl+R (or r) to repeat with the same config, Enter to wait for the next command:\n> ", last.RequestID)

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
