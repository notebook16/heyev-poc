package config

type DeliveryMode string

const (
	DeliveryModeA DeliveryMode = "A" // Retain + Message Expiry
	DeliveryModeB DeliveryMode = "B" // Session queue + Message Expiry (retain=false)
)

func (m DeliveryMode) Label() string {
	switch m {
	case DeliveryModeA:
		return "Option A (Retain + Message Expiry)"
	case DeliveryModeB:
		return "Option B (Session queue + Message Expiry)"
	default:
		return string(m)
	}
}

func (c *Config) EffectiveRetain() bool {
	if c.DeliveryMode == DeliveryModeB {
		return false
	}
	return c.Retain
}
