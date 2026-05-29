package service

import (
	"context"
	"testing"

	"LoopGuard/internal/executor"
	"LoopGuard/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApproveTriggersExecution(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)

	tk, err := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"env": "prod"}})
	require.NoError(t, err)
	require.Equal(t, model.StatusPendingApproval, tk.Status)

	fe.result = &executor.ExecResult{ExitCode: 0, Stdout: "deployed"}
	out, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusDone, out.Status)
	assert.NotNil(t, out.ApprovedBy)
	assert.Equal(t, p.ApproverID, *out.ApprovedBy)
}

func TestApproveWrongUserRejected(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)
	tk, _ := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"env": "prod"}})

	_, err := svc.Approve(context.Background(), tk.ID, p.ApproverID+999)
	require.Error(t, err)
}

func TestApproveExecFailed(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)
	tk, _ := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"env": "prod"}})

	fe.result = &executor.ExecResult{ExitCode: 5, Stderr: "boom"}
	out, err := svc.Approve(context.Background(), tk.ID, p.ApproverID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusExecFailed, out.Status)
}

func TestReject(t *testing.T) {
	fe := &fakeExecutor{result: &executor.ExecResult{ExitCode: 0, Stdout: "DRYRUN-OK"}}
	svc, s := newTicketService(t, fe)
	p := seedProgram(t, s, `{"env":"string"}`)
	tk, _ := svc.Submit(context.Background(), SubmitInput{
		Project: "demo", Name: "deploy", APIKeyID: 1, Args: map[string]any{"env": "prod"}})

	out, err := svc.Reject(tk.ID, p.ApproverID, "太危险")
	require.NoError(t, err)
	assert.Equal(t, model.StatusRejected, out.Status)
	assert.Equal(t, "太危险", out.RejectReason)
}
