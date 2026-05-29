package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanTransition(t *testing.T) {
	assert.True(t, CanTransition(StatusPendingDryrun, StatusPendingApproval))
	assert.True(t, CanTransition(StatusPendingDryrun, StatusDryrunFailed))
	assert.True(t, CanTransition(StatusPendingApproval, StatusApproved))
	assert.True(t, CanTransition(StatusPendingApproval, StatusRejected))
	assert.True(t, CanTransition(StatusApproved, StatusExecuting))
	assert.True(t, CanTransition(StatusExecuting, StatusDone))
	assert.True(t, CanTransition(StatusExecuting, StatusExecFailed))

	// 非法流转
	assert.False(t, CanTransition(StatusDone, StatusExecuting))
	assert.False(t, CanTransition(StatusRejected, StatusApproved))
	assert.False(t, CanTransition(StatusPendingDryrun, StatusDone))
}
