package btype

// subscribe type
type SubscribeType int

const (
	NormalSubscribe           = SubscribeType(1)
	SequentialSubscribe       = SubscribeType(2)
	DelaySubscribe            = SubscribeType(3)
	SequentialByLockSubscribe = SubscribeType(4)
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
	NORMAL           MoodType = "normal"
	DELAY            MoodType = "delay"
	SEQUENCE         MoodType = "sequential"
	SEQUENCE_BY_LOCK MoodType = "sequential_by_lock"
)
