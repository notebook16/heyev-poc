package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	fmt.Println("Starting IoT Simulator POC")

	// --------------------------------------------------
	// 1. Load Amazon Root CA
	// --------------------------------------------------

	caCert, err := os.ReadFile(
		"../certs/simulator/AmazonRootCA1.pem",
	)
	if err != nil {
		fmt.Println("Failed to read CA Certificate:", err)
		os.Exit(1)
	}

	caPool := x509.NewCertPool()

	if !caPool.AppendCertsFromPEM(caCert) {
		fmt.Println("Failed to add CA Certificate")
		os.Exit(1)
	}

	// --------------------------------------------------
	// 2. Load simulator certificate + private key
	// --------------------------------------------------

	
		
	cert, err := tls.LoadX509KeyPair(
    		"../certs/simulator/a75bd3d3cf1ed6d465c3114ee171b232cef5cb6e5e0eb9a01dd83034001d9879-certificate.pem.crt",
    		"../certs/simulator/a75bd3d3cf1ed6d465c3114ee171b232cef5cb6e5e0eb9a01dd83034001d9879-private.pem.key",
	)
	

	if err != nil {
		fmt.Println("Failed to load simulator certificate:", err)
		os.Exit(1)
	}

	// --------------------------------------------------
	// 3. TLS configuration
	// --------------------------------------------------

	tlsConfig := &tls.Config{
		RootCAs:      caPool,
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		ServerName:   os.Getenv("IOT_ENDPOINT"),
	}

	// --------------------------------------------------
	// 4. MQTT configuration
	// --------------------------------------------------

	opts := mqtt.NewClientOptions()

	opts.AddBroker(
		"ssl://" + os.Getenv("IOT_ENDPOINT") + ":8883",
	)

	opts.SetClientID("iot-simulator-poc")

	// MQTT 3.1.1
	opts.SetProtocolVersion(4)

	opts.SetTLSConfig(tlsConfig)

	// --------------------------------------------------
	// 5. Command handler
	// --------------------------------------------------

	opts.SetDefaultPublishHandler(
		func(client mqtt.Client, msg mqtt.Message) {

			fmt.Println()
			fmt.Println("========== COMMAND RECEIVED ==========")
			fmt.Println("Topic:", msg.Topic())
			fmt.Println("QoS:", msg.Qos())
			fmt.Println("Payload:", string(msg.Payload()))
			fmt.Println("======================================")

			// For our POC, this simulator represents
			// device 6264.

			ackTopic := "heyev/v1/devices/6264/ack"

			ackPayload := `{
				"device_id": "6264",
				"status": "ACKNOWLEDGED",
				"message": "Command received by simulator"
			}`

			token := client.Publish(
				ackTopic,
				0,
				false,
				ackPayload,
			)

			if token.Wait() && token.Error() != nil {
				fmt.Println("Failed to publish ACK:", token.Error())
				return
			}

			fmt.Println("ACK published to:", ackTopic)
		},
	)

	// --------------------------------------------------
	// 6. Create MQTT client
	// --------------------------------------------------

	client := mqtt.NewClient(opts)

	fmt.Println("Connecting to AWS IoT Core...")
	fmt.Println("Endpoint:", os.Getenv("IOT_ENDPOINT"))
	fmt.Println("Client ID: iot-simulator-poc")

	token := client.Connect()

	if token.Wait() && token.Error() != nil {
		fmt.Println("Connection Failed:", token.Error())
		os.Exit(1)
	}

	fmt.Println("Connected to AWS IoT Core Successfully!")

	// --------------------------------------------------
	// 7. Subscribe to commands
	// --------------------------------------------------

	commandTopic := "heyev/v1/devices/+/commands"

	fmt.Println("Subscribing to:", commandTopic)

	token = client.Subscribe(
		commandTopic,
		0,
		nil,
	)

	if token.Wait() && token.Error() != nil {
		fmt.Println("Subscription Failed:", token.Error())
		client.Disconnect(250)
		os.Exit(1)
	}

	fmt.Println("Successfully subscribed to command topic!")
	fmt.Println("Simulator is waiting for commands...")
	fmt.Println("Press Ctrl+C to stop.")

	// --------------------------------------------------
	// 8. Keep simulator alive
	// --------------------------------------------------

	sigChan := make(chan os.Signal, 1)

	signal.Notify(
		sigChan,
		os.Interrupt,
		syscall.SIGTERM,
	)

	<-sigChan

	fmt.Println()
	fmt.Println("Stopping IoT Simulator...")

	client.Disconnect(250)

	fmt.Println("Disconnected from AWS IoT Core.")
}
