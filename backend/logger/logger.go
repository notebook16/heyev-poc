package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

type Logger struct {
	out *log.Logger
}

func New() *Logger {
	return &Logger{out: log.New(os.Stdout, "", 0)}
}

func (l *Logger) SetOutput(w io.Writer) {
	l.out = log.New(w, "", 0)
}

func (l *Logger) log(prefix, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	l.out.Printf("%s %s", prefix, msg)
}

func (l *Logger) Config(format string, args ...any)  { l.log("[CONFIG]", format, args...) }
func (l *Logger) MQTT(format string, args ...any)    { l.log("[MQTT]", format, args...) }
func (l *Logger) TLS(format string, args ...any)     { l.log("[TLS]", format, args...) }
func (l *Logger) Connect(format string, args ...any) { l.log("[CONNECT]", format, args...) }
func (l *Logger) Disconnect(format string, args ...any) {
	l.log("[DISCONNECT]", format, args...)
}
func (l *Logger) Reconnect(format string, args ...any) { l.log("[RECONNECT]", format, args...) }
func (l *Logger) Subscribe(format string, args ...any) { l.log("[SUBSCRIBE]", format, args...) }
func (l *Logger) Publish(format string, args ...any)   { l.log("[PUBLISH]", format, args...) }
func (l *Logger) PubAck(format string, args ...any)    { l.log("[PUBACK]", format, args...) }
func (l *Logger) Command(format string, args ...any)   { l.log("[COMMAND]", format, args...) }
func (l *Logger) Ack(format string, args ...any)       { l.log("[ACK]", format, args...) }
func (l *Logger) Idempotency(format string, args ...any) {
	l.log("[IDEMPOTENCY]", format, args...)
}
func (l *Logger) Expiry(format string, args ...any) { l.log("[EXPIRY]", format, args...) }
func (l *Logger) State(format string, args ...any)  { l.log("[STATE]", format, args...) }
func (l *Logger) Error(format string, args ...any)  { l.log("[ERROR]", format, args...) }
