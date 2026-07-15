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

const (
	// SequenceQueueCodeScheduled means the first customer message created a scheduler token.
	SequenceQueueCodeScheduled = "SCHEDULED"
	// SequenceQueueCodeQueued means a message joined an existing customer queue.
	SequenceQueueCodeQueued = "QUEUED"
	// SequenceQueueCodeFull means the partition reached its pending-message capacity.
	SequenceQueueCodeFull = "FULL"
	// SequenceQueueCodeAcquired means a consumer obtained exclusive ownership of the queue head.
	SequenceQueueCodeAcquired = "ACQUIRED"
	// SequenceQueueCodeRenewed means the current consumer extended its ownership lease.
	SequenceQueueCodeRenewed = "RENEWED"
	// SequenceQueueCodeBusy means another unexpired acquisition owns the customer queue.
	SequenceQueueCodeBusy = "BUSY"
	// SequenceQueueCodeSuccessor means finalization scheduled the next customer message.
	SequenceQueueCodeSuccessor = "SUCCESSOR"
	// SequenceQueueCodeEmpty means finalization removed the empty customer queue and its state.
	SequenceQueueCodeEmpty = "EMPTY"
	// SequenceQueueCodeStaleAcquisition means the acquisition no longer matches the current owner.
	SequenceQueueCodeStaleAcquisition = "STALE_ACQUISITION"
	// SequenceQueueCodeStaleCleaned means an obsolete scheduler token was removed.
	SequenceQueueCodeStaleCleaned = "STALE_CLEANED"
	// SequenceQueueCodeOrphanCleaned means a scheduler token without customer state was removed.
	SequenceQueueCodeOrphanCleaned = "ORPHAN_CLEANED"
	// SequenceQueueCodeEmptyCleaned means inconsistent empty-list state was repaired.
	SequenceQueueCodeEmptyCleaned = "EMPTY_CLEANED"
	// SequenceQueueCodeIsolated means inconsistent queue data was moved out of scheduling.
	SequenceQueueCodeIsolated = "ISOLATED"
	// SequenceQueueCodeNotPending means the scheduler token is absent from the pending-entry list.
	SequenceQueueCodeNotPending = "NOT_PENDING"
	// SequenceQueueCodePELOwnerMismatch means the scheduler token belongs to another Redis consumer.
	SequenceQueueCodePELOwnerMismatch = "PEL_OWNER_MISMATCH"
	// SequenceQueueCodeHeadMismatch means the finalized message is no longer the customer queue head.
	SequenceQueueCodeHeadMismatch = "HEAD_MISMATCH"
)
