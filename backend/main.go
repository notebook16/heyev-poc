package main

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	

	mqtt "github.com/eclipse/paho.mqtt.golang"
) 



func main () { 
	fmt.Println("starting Heyev backend poc")

	//load certificate
	caCert , err := os.ReadFile("../certs/backend/AmazonRootCA1.pem")
	if err != nil {
	fmt.Println("Failed to read CA Certificate", err)
	os.Exit(1)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert){
		fmt.Println("Failed to add CA Certificate")
		os.Exit(1)
	}

	//device certificate and private key
	cert, err := tls.LoadX509KeyPair(
	
		"../certs/backend/46fe5a021abb1fa075d856403d9c3ce6ce183cfe15a5eb885e9d335b2db21849-certificate.pem.crt",
		"../certs/backend/46fe5a021abb1fa075d856403d9c3ce6ce183cfe15a5eb885e9d335b2db21849-private.pem.key",
	)

	if err != nil { 

	fmt.Println("Failed to load certificate: ", err)
	os.Exit(1)

	 }

	tlsConfig := &tls.Config{

		RootCAs: caPool,
		Certificates : []tls.Certificate{cert},
		MinVersion : tls.VersionTLS12,
		ServerName:   os.Getenv("IOT_ENDPOINT"),
		InsecureSkipVerify: false,
	}

 	opts := mqtt.NewClientOptions()

	opts.AddBroker("ssl://" + os.Getenv("IOT_ENDPOINT") + ":8883")
	opts.SetClientID("heyev-backend-poc")
	opts.SetTLSConfig(tlsConfig)
	opts.SetProtocolVersion(4) 

	

	mqtt.ERROR = log.New(os.Stderr, "ERROR ", log.LstdFlags)
	mqtt.CRITICAL = log.New(os.Stderr, "CRITICAL ", log.LstdFlags)
	mqtt.WARN = log.New(os.Stderr, "WARN ", log.LstdFlags)
	mqtt.DEBUG = log.New(os.Stderr, "DEBUG ", log.LstdFlags)

	client := mqtt.NewClient(opts)

	fmt.Println("connecting to AWS IoT Core....")

	fmt.Println("Endpoint:", os.Getenv("IOT_ENDPOINT"))
	fmt.Println("Client ID: heyev-backend-poc")
	fmt.Println("Connecting...")

	token := client.Connect()

	if token.Wait() && token.Error() != nil {
	fmt.Println("COnnection Failed:" , token.Error())
	os.Exit(1)
	}

	fmt.Println("connected to AWS IoT Core Successfully!")

	
	// --------------------------------------------------
	// 7. Subscribe to ACK topic
	// --------------------------------------------------

	ackTopic := "heyev/v1/devices/+/ack"

	fmt.Println("subscribe to: ", ackTopic)

	token = client.Subscribe(
		ackTopic,
		0,
		func(client mqtt.Client , msg mqtt.Message) {
	
			fmt.Println()
			fmt.Println("========== ACK RECEIVED ==========")
			fmt.Println("Topic:", msg.Topic())
			fmt.Println("QoS:", msg.Qos())
			fmt.Println("Payload:", string(msg.Payload()))
			fmt.Println("==================================")
			fmt.Println()	
		},	

	)

	
	if token.Wait() && token.Error() != nil {
		fmt.Println("Subscription Failed:", token.Error())
		client.Disconnect(250)
		os.Exit(1)
	}

	fmt.Println("Successfully subscribed to ACK topic!")
	fmt.Println("Backend is now listening for ACKs...")
	fmt.Println("Press Ctrl+C to stop.")
	
	commandTopic := "heyev/v1/devices/6264/commands"

	fmt.Println("Publishing command to:", commandTopic)

	payload := `{
		"device_id": "6264",
		"command": "TEST",
		"value": "hello"
	}`

	token = client.Publish(
	 commandTopic,
	 0, // QoS 0
	 false, // retain
	 payload,
	)

	if token.Wait() && token.Error() != nil {
		fmt.Println("Publish Failed:", token.Error())
		client.Disconnect(250)
		os.Exit(1)
	}

	fmt.Println("Command published successfully!")

	// --------------------------------------------------
	// 8. Keep process alive
	// --------------------------------------------------

	sigChan := make(chan os.Signal,1)

	signal.Notify(
		sigChan,
		os.Interrupt,
		syscall.SIGTERM,
	)
	
	
	<-sigChan
	
	fmt.Println()
	fmt.Println("Stopping HeyEV Backend POC...")

	client.Disconnect(250)

	fmt.Println("Disconnected from AWS IoT Core.")

	


<<<<<<< Updated upstream
=======
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
	log.Config("Delivery mode: %s", cfg.DeliveryMode.Label())
	log.Config("MQTT Version: 5")
	log.Config("QoS: %d", cfg.QoS)
	log.Config("Retain: %t (effective on publish: %t)", cfg.Retain, cfg.EffectiveRetain())
	log.Config("Session expiry: %ds", cfg.SessionExpirySec)
	log.Config("Persistent session: %t", cfg.PersistentSession)
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
>>>>>>> Stashed changes
}
