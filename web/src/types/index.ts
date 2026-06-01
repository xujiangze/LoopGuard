export type Role = "user" | "admin"

export type TicketStatus =
  | "pending_dryrun"
  | "dryrun_failed"
  | "pending_approval"
  | "approved"
  | "executing"
  | "done"
  | "exec_failed"
  | "rejected"

export interface User {
  id: number
  username: string
  role: Role
  created_at: string
}

export interface Ticket {
  id: number
  program_id: number
  args: Record<string, unknown>
  status: TicketStatus
  submitted_by: number
  approver_id: number
  dryrun_output: string
  approved_by: number | null
  approved_at: string | null
  reject_reason: string
  created_at: string
  updated_at: string
}

export interface Program {
  id: number
  project: string
  name: string
  binary_path: string
  interpreter: string
  help_text: string
  params_schema: Record<string, unknown> | null
  approver_id: number
  timeout_sec: number
  supports_dryrun: boolean
  enabled: boolean
  created_at: string
}

export interface APIKey {
  id: number
  name: string
  enabled: boolean
  created_at: string
}

export interface Execution {
  id: number
  ticket_id: number
  kind: "dryrun" | "real"
  command: string
  exit_code: number
  stdout: string
  stderr: string
  duration_ms: number
  started_at: string | null
  finished_at: string | null
}

export const TICKET_STATUS_LABELS: Record<TicketStatus, string> = {
  pending_dryrun: "Dry-run 中",
  dryrun_failed: "Dry-run 失败",
  pending_approval: "待审批",
  approved: "已批准",
  executing: "执行中",
  done: "已完成",
  exec_failed: "执行失败",
  rejected: "已驳回",
}
