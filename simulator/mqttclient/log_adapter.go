package mqttclient

import (
	"fmt"
	"strings"

	"iot-simulator-poc/logger"
)

type stdAdapter struct {
	log *logger.Logger
}

func (a stdAdapter) Println(v ...interface{}) {
	msg := fmt.Sprint(v...)
	if noisyMQTTLog(msg) {
		return
	}
	a.log.MQTT("%s", msg)
}

func (a stdAdapter) Printf(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	if noisyMQTTLog(msg) {
		return
	}
	a.log.MQTT("%s", strings.TrimRight(msg, "\n"))
}

func noisyMQTTLog(msg string) bool {
	switch {
	case strings.Contains(msg, "PINGRESP"),
		strings.Contains(msg, "PINGREQ"),
		strings.Contains(msg, "PingHandler"),
		strings.Contains(msg, "queue AwaitConnection"),
		strings.Contains(msg, "queue got connection"):
		return true
	default:
		return false
	}
}
