package mqttclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	paholog "github.com/eclipse/paho.golang/paho/log"

	"iot-simulator-poc/config"
	"iot-simulator-poc/logger"
)

type Client struct {
	cfg          *config.Config
	log          *logger.Logger
	cm           *autopaho.ConnectionManager
	subTopic     string
	onMessage    func(*paho.Publish)
	mu           sync.Mutex
	firstUp      bool
	firstSubDone chan struct{}
	firstSubOnce sync.Once
	firstSubErr  error
}

func New(cfg *config.Config, log *logger.Logger, onMessage func(*paho.Publish)) *Client {
	return &Client{
		cfg:          cfg,
		log:          log,
		onMessage:    onMessage,
		firstSubDone: make(chan struct{}),
	}
}

func (c *Client) SetMessageHandler(fn func(*paho.Publish)) {
	c.onMessage = fn
}

func (c *Client) Connect(ctx context.Context, tlsCfg *tls.Config, subscribeTopic string) error {
	c.subTopic = subscribeTopic

	serverURL, err := url.Parse(fmt.Sprintf("tls://%s:8883", c.cfg.Endpoint))
	if err != nil {
		return fmt.Errorf("parse broker URL: %w", err)
	}

	cleanStart := !c.cfg.PersistentSession
	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{serverURL},
		TlsCfg:                        tlsCfg,
		KeepAlive:                     30,
		CleanStartOnInitialConnection: cleanStart,
		SessionExpiryInterval:         c.cfg.SessionExpirySec,
		ConnectTimeout:                30 * time.Second,
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			c.onConnectionUp(cm, connAck)
		},
		OnConnectionDown: func() bool {
			c.log.Disconnect("SIMULATOR CONNECTION LOST")
			if c.cfg.AutoReconnect {
				c.log.Reconnect("SIMULATOR RECONNECTING")
			}
			return c.cfg.AutoReconnect
		},
		OnConnectError: func(err error) {
			c.log.Reconnect("Connection attempt failed: %v", err)
		},
		ClientConfig: paho.ClientConfig{
			ClientID: c.cfg.ClientID,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					if c.onMessage != nil && pr.Packet != nil {
						c.onMessage(pr.Packet)
					}
					return true, nil
				},
			},
		},
	}

	if c.cfg.Debug {
		adapter := stdAdapter{log: c.log}
		cliCfg.Debug = adapter
		cliCfg.PahoDebug = adapter
		cliCfg.Errors = adapter
		cliCfg.PahoErrors = adapter
	} else {
		noop := paholog.NOOPLogger{}
		cliCfg.Debug = noop
		cliCfg.PahoDebug = noop
		cliCfg.Errors = log.New(os.Stderr, "[MQTT-ERR] ", log.LstdFlags)
		cliCfg.PahoErrors = log.New(os.Stderr, "[PAHO-ERR] ", log.LstdFlags)
	}

	cm, err := autopaho.NewConnection(ctx, cliCfg)
	if err != nil {
		return fmt.Errorf("create connection: %w", err)
	}

	c.cm = cm

	c.log.Connect("Connecting to AWS IoT Core...")
	c.log.Connect("Endpoint: %s", c.cfg.Endpoint)
	c.log.Connect("Client ID: %s", c.cfg.ClientID)
	c.log.Session("Delivery mode: %s", c.cfg.DeliveryMode.Label())
	c.log.Session("Clean Start: %t", cleanStart)
	c.log.Session("Session expiry: %ds", c.cfg.SessionExpirySec)
	c.log.Session("Persistent session: %t", c.cfg.PersistentSession)
	if c.cfg.DeliveryMode == config.DeliveryModeB {
		c.log.Session("Option B: QoS 1 subscribe required for offline command queue")
	}

	if err := cm.AwaitConnection(ctx); err != nil {
		return fmt.Errorf("await connection: %w", err)
	}

	if c.subTopic == "" {
		return nil
	}

	select {
	case <-ctx.Done():
		return fmt.Errorf("await subscribe: %w", ctx.Err())
	case <-c.firstSubDone:
		c.mu.Lock()
		err := c.firstSubErr
		c.mu.Unlock()
		if err != nil {
			return fmt.Errorf("subscribe %s: %w", c.subTopic, err)
		}
		return nil
	}
}

func (c *Client) signalFirstSub(err error) {
	c.firstSubOnce.Do(func() {
		c.mu.Lock()
		c.firstSubErr = err
		c.mu.Unlock()
		close(c.firstSubDone)
	})
}

func (c *Client) onConnectionUp(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
	c.mu.Lock()
	reconnect := c.firstUp
	c.firstUp = true
	c.mu.Unlock()

	sessionPresent := false
	if connAck != nil {
		sessionPresent = connAck.SessionPresent
	}

	if reconnect {
		c.log.Connect("SIMULATOR RECONNECTED")
		c.log.Session("SessionPresent (CONNACK): %t", sessionPresent)
		if c.cfg.DeliveryMode == config.DeliveryModeB && sessionPresent {
			c.log.Session("Option B: broker may deliver queued commands after reconnect")
		}
	} else {
		c.log.Connect("Connected to AWS IoT Core")
		c.log.Session("SessionPresent (CONNACK): %t", sessionPresent)
	}

	if c.subTopic == "" {
		c.signalFirstSub(nil)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	subQoS := c.cfg.CommandSubscribeQoS()
	_, err := cm.Subscribe(ctx, &paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{
			{Topic: c.subTopic, QoS: subQoS},
		},
	})
	if err != nil {
		c.log.Error("Subscribe failed for %s: %v", c.subTopic, err)
		c.signalFirstSub(err)
		return
	}

	c.log.Subscribe("Topic: %s", c.subTopic)
	c.log.Subscribe("QoS: %d", subQoS)
	if reconnect {
		c.log.Subscribe("COMMAND SUBSCRIPTION RESTORED")
	} else {
		c.log.Subscribe("Successful")
	}
	c.signalFirstSub(nil)
}

func (c *Client) Manager() *autopaho.ConnectionManager {
	return c.cm
}

func (c *Client) PublishACK(ctx context.Context, topic string, payload []byte) error {
	c.log.Ack("Publishing ACK")
	c.log.Ack("Topic: %s", topic)
	c.log.Ack("QoS: %d", c.cfg.QoS)
	c.log.Ack("Retain: %t", c.cfg.Retain)
	if c.cfg.Retain {
		c.log.Ack("NOTE: RETAIN=true — retained ACK message")
	}
	c.log.Ack("Payload: %s", string(payload))

	pub := &paho.Publish{
		Topic:   topic,
		QoS:     c.cfg.QoS,
		Retain:  c.cfg.Retain,
		Payload: payload,
	}

	resp, err := c.cm.Publish(ctx, pub)
	if err != nil {
		c.log.Error("ACK publish failed: %v", err)
		return err
	}

	if c.cfg.QoS >= 1 && resp != nil {
		c.log.PubAck("MQTT PUBACK received for ACK publish (reason=%d)", resp.ReasonCode)
	}

	c.log.Ack("ACK published successfully")
	return nil
}

func (c *Client) Disconnect(ctx context.Context) error {
	if c.cm == nil {
		return nil
	}
	return c.cm.Disconnect(ctx)
}
