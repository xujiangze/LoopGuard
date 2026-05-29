package model

type TicketStatus string

const (
	StatusPendingDryrun   TicketStatus = "pending_dryrun"
	StatusDryrunFailed    TicketStatus = "dryrun_failed"
	StatusPendingApproval TicketStatus = "pending_approval"
	StatusApproved        TicketStatus = "approved"
	StatusExecuting       TicketStatus = "executing"
	StatusDone            TicketStatus = "done"
	StatusExecFailed      TicketStatus = "exec_failed"
	StatusRejected        TicketStatus = "rejected"
)

var allowed = map[TicketStatus][]TicketStatus{
	StatusPendingDryrun:   {StatusPendingApproval, StatusDryrunFailed},
	StatusPendingApproval: {StatusApproved, StatusRejected},
	StatusApproved:        {StatusExecuting},
	StatusExecuting:       {StatusDone, StatusExecFailed},
}

func CanTransition(from, to TicketStatus) bool {
	for _, s := range allowed[from] {
		if s == to {
			return true
		}
	}
	return false
}
