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

export interface TicketListItem {
  id: number
  program_id: number
  program_project: string
  program_name: string
  args: string[]
  status: TicketStatus
  submitted_by: number
  submitted_by_name: string
  approver_id: number
  approved_by: number | null
  approved_at: string | null
  reject_reason: string
  created_at: string
  updated_at: string
}

export interface Ticket {
  id: number
  program_id: number
  args: string[]
  status: TicketStatus
  submitted_by: number
  approver_id: number
  dryrun_output: string
  exec_output: string
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
  entry_file: string
  interpreter: string
  help_text: string
  approver_id: number
  timeout_sec: number
  supports_dryrun: boolean
  enabled: boolean
  current_version: number
  created_at: string
  updated_at: string
}

export interface ProgramVersion {
  id: number
  program_id: number
  version: number
  entry_file: string
  interpreter: string
  help_text: string
  is_rollback: boolean
  created_by: string
  created_at: string
}

export interface ProgramFileInfo {
  name: string
  size: number
  is_entry: boolean
  mod_time: string
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

export interface WebhookConfig {
  id: number
  program_id: number
  name: string
  url: string
  enabled: boolean
  event_types: string
  created_at: string
  updated_at: string
}

export interface WebhookDelivery {
  id: number
  webhook_id: number
  ticket_id: number
  event_type: string
  status_code: number
  response: string
  delivered_at: string | null
}

export type WebhookEventType =
  | "ticket.pending_approval"
  | "ticket.dryrun_failed"
  | "ticket.done"
  | "ticket.exec_failed"
  | "ticket.rejected"

export const WEBHOOK_EVENT_LABELS: Record<WebhookEventType, string> = {
  "ticket.pending_approval": "待审批",
  "ticket.dryrun_failed": "Dry-run 失败",
  "ticket.done": "执行完成",
  "ticket.exec_failed": "执行失败",
  "ticket.rejected": "已驳回",
}
