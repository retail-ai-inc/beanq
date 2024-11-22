package bstatus

type (
	FlagInfo = string
	LevelMsg = string
	Status   = string
)

const (
	SuccessInfo FlagInfo = "success"
	FailedInfo  FlagInfo = "failed"

	StatusPrepare    Status = "prepare"
	StatusPublished  Status = "published"
	StatusPending    Status = "pending"
	StatusReceived   Status = "received"
	StatusSuccess    Status = "success"
	StatusFailed     Status = "failed"
	StatusDeadLetter Status = "dead_letter"

	ErrLevel  LevelMsg = "error"
	InfoLevel LevelMsg = "info"
)
