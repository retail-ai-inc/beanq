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

type LogType = string

const (
	Dlq       LogType = "dlq"   // deadLetter message
	Logic     LogType = "logic" // logic message : normal,delay,sequential
	Operation LogType = "opt"   // UI access log
)
