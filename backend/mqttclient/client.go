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

	"heyev-backend-poc/config"
	"heyev-backend-poc/logger"
)

type Client struct {
	cfg       *config.Config
	log       *logger.Logger
	cm        *autopaho.ConnectionManager
	subTopic  string
	onMessage func(*paho.Publish)
	mu        sync.Mutex
	firstUp   bool
}

type stdAdapter struct {
	log *logger.Logger
}

func (a stdAdapter) Println(v ...interface{}) {
	a.log.MQTT("%s", fmt.Sprint(v...))
}

func (a stdAdapter) Printf(format string, v ...interface{}) {
	a.log.MQTT(format, v...)
}

func New(cfg *config.Config, log *logger.Logger, onMessage func(*paho.Publish)) *Client {
	return &Client{
		cfg:       cfg,
		log:       log,
		onMessage: onMessage,
	}
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
			c.log.Disconnect("CONNECTION LOST")
			if c.cfg.AutoReconnect {
				c.log.Reconnect("RECONNECTING")
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

	if err := cm.AwaitConnection(ctx); err != nil {
		return fmt.Errorf("await connection: %w", err)
	}

	return nil
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
		c.log.Connect("RECONNECTED")
		c.log.Session("SessionPresent (CONNACK): %t", sessionPresent)
		c.log.Subscribe("SUBSCRIBING AFTER RECONNECT")
	} else {
		c.log.Connect("Connected to AWS IoT Core")
		c.log.Session("SessionPresent (CONNACK): %t", sessionPresent)
	}

	if c.subTopic == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := cm.Subscribe(ctx, &paho.Subscribe{
		Subscriptions: []paho.SubscribeOptions{
			{Topic: c.subTopic, QoS: c.cfg.QoS},
		},
	})
	if err != nil {
		c.log.Error("Subscribe failed for %s: %v", c.subTopic, err)
		return
	}

	c.log.Subscribe("Topic: %s", c.subTopic)
	c.log.Subscribe("QoS: %d", c.cfg.QoS)
	if reconnect {
		c.log.Subscribe("SUBSCRIPTION RESTORED")
	} else {
		c.log.Subscribe("Successful")
	}
}

func (c *Client) Manager() *autopaho.ConnectionManager {
	return c.cm
}

func (c *Client) Disconnect(ctx context.Context) error {
	if c.cm == nil {
		return nil
	}
	return c.cm.Disconnect(ctx)
}
