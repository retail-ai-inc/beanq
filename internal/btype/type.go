package btype

// subscribe type
type SubscribeType int

const (
	NormalSubscribe     = SubscribeType(1)
	SequentialSubscribe = SubscribeType(2)
	DelaySubscribe      = SubscribeType(3)
)

// MoodType message type
type MoodType string

func (m MoodType) String() string {
	return string(m)
}

func (m MoodType) MarshalBinary() ([]byte, error) {
	return []byte(m), nil
}

const (
	NORMAL     MoodType = "normal"
	DELAY      MoodType = "delay"
	SEQUENTIAL MoodType = "sequential"
)
