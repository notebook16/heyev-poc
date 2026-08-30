package config

type DeliveryMode string

const (
	DeliveryModeA DeliveryMode = "A" // Retain-friendly; broker stores last command on topic
	DeliveryModeB DeliveryMode = "B" // Session queue; subscribe QoS 1 + persistent session
)

func (m DeliveryMode) Label() string {
	switch m {
	case DeliveryModeA:
		return "Option A (Retain + Message Expiry on commands from backend)"
	case DeliveryModeB:
		return "Option B (Session queue + Message Expiry on commands from backend)"
	default:
		return string(m)
	}
}

func (c *Config) CommandSubscribeQoS() byte {
	return c.QoS
}
