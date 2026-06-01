import { useEffect, useState } from "react"
import { useParams, useNavigate, Link } from "react-router-dom"
import { api } from "@/lib/api"
import type { Ticket, Execution, APIKey } from "@/types"
import { TICKET_STATUS_LABELS } from "@/types"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"

export function TicketDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [ticket, setTicket] = useState<Ticket | null>(null)
  const [executions, setExecutions] = useState<Execution[]>([])
  const [apiKeyMap, setApiKeyMap] = useState<Map<number, string>>(new Map())
  const [error, setError] = useState("")
  const [rejectOpen, setRejectOpen] = useState(false)
  const [rejectReason, setRejectReason] = useState("")
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    api
      .get<APIKey[]>("/api-keys")
      .then((keys) => {
        const m = new Map<number, string>()
        for (const k of keys) m.set(k.id, k.name)
        setApiKeyMap(m)
      })
      .catch(() => {})
  }, [])

  useEffect(() => {
    if (!id) return
    api
      .get<Ticket>(`/tickets/${id}`)
      .then(setTicket)
      .catch((err) => setError(err.message))
  }, [id])

  useEffect(() => {
    if (!ticket) return
    api
      .get<Execution[]>(`/tickets/${ticket.id}/executions`)
      .then(setExecutions)
      .catch(() => setExecutions([]))
  }, [ticket])

  const handleApprove = async () => {
    if (!ticket) return
    setSubmitting(true)
    try {
      await api.post(`/tickets/${ticket.id}/approve`)
      navigate("/", { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : "审批失败")
    } finally {
      setSubmitting(false)
    }
  }

  const handleReject = async () => {
    if (!ticket) return
    setSubmitting(true)
    try {
      await api.post(`/tickets/${ticket.id}/reject`, { reason: rejectReason })
      navigate("/", { replace: true })
    } catch (err) {
      setError(err instanceof Error ? err.message : "驳回失败")
    } finally {
      setSubmitting(false)
      setRejectOpen(false)
    }
  }

  if (error && !ticket) {
    return (
      <div className="flex flex-col items-center justify-center py-20 space-y-4">
        <p className="text-muted-foreground">{error}</p>
        <Link to="/">
          <Button variant="outline">返回列表</Button>
        </Link>
      </div>
    )
  }

  if (!ticket) {
    return <p className="text-muted-foreground py-8 text-center">加载中...</p>
  }

  const isPendingApproval = ticket.status === "pending_approval"
  const realExec = executions.find((e) => e.kind === "real")

  return (
    <div className="space-y-6 max-w-3xl">
      <div className="flex items-center gap-3">
        <Link to="/">
          <Button variant="ghost" size="sm">
            ← 返回
          </Button>
        </Link>
        <h1 className="text-2xl font-semibold">工单 #{ticket.id}</h1>
        <Badge>{TICKET_STATUS_LABELS[ticket.status]}</Badge>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}

      {/* 基本信息 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">基本信息</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">程序 ID</span>
            <span>{ticket.program_id}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">提交时间</span>
            <span>{new Date(ticket.created_at).toLocaleString("zh-CN")}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">提交来源</span>
            <span>{apiKeyMap.get(ticket.submitted_by) ?? `Key #${ticket.submitted_by}`}</span>
          </div>
        </CardContent>
      </Card>

      {/* AI 提交参数 */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">AI 提交参数</CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="bg-muted rounded-md p-3 text-sm overflow-auto">
            {JSON.stringify(ticket.args, null, 2)}
          </pre>
        </CardContent>
      </Card>

      {/* Dry-run 输出 */}
      {ticket.dryrun_output && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Dry-run 输出</CardTitle>
          </CardHeader>
          <CardContent>
            <pre className="bg-muted rounded-md p-3 text-sm overflow-auto whitespace-pre-wrap">
              {ticket.dryrun_output}
            </pre>
          </CardContent>
        </Card>
      )}

      {/* 已驳回：驳回原因 */}
      {ticket.status === "rejected" && ticket.reject_reason && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">驳回原因</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm">{ticket.reject_reason}</p>
          </CardContent>
        </Card>
      )}

      {/* 已完成/执行失败：执行结果 */}
      {(ticket.status === "done" || ticket.status === "exec_failed") && realExec && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">执行结果</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex gap-6">
              <div>
                <span className="text-muted-foreground">退出码：</span>
                <span className={realExec.exit_code !== 0 ? "text-destructive" : ""}>
                  {realExec.exit_code}
                </span>
              </div>
              <div>
                <span className="text-muted-foreground">耗时：</span>
                <span>{realExec.duration_ms}ms</span>
              </div>
            </div>
            {realExec.stdout && (
              <div>
                <p className="text-muted-foreground mb-1">stdout：</p>
                <pre className="bg-muted rounded-md p-3 text-sm overflow-auto whitespace-pre-wrap">
                  {realExec.stdout}
                </pre>
              </div>
            )}
            {realExec.stderr && (
              <div>
                <p className="text-muted-foreground mb-1">stderr：</p>
                <pre className="bg-muted rounded-md p-3 text-sm overflow-auto whitespace-pre-wrap text-destructive">
                  {realExec.stderr}
                </pre>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* 审批操作区 */}
      {isPendingApproval && (
        <>
          <Separator />
          <div className="flex gap-3">
            <Button onClick={handleApprove} disabled={submitting}>
              {submitting ? "处理中..." : "批准执行"}
            </Button>
            <Button variant="destructive" onClick={() => setRejectOpen(true)} disabled={submitting}>
              驳回
            </Button>
          </div>
        </>
      )}

      {/* 驳回弹窗 */}
      <Dialog open={rejectOpen} onOpenChange={setRejectOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>驳回工单</DialogTitle>
          </DialogHeader>
          <Textarea
            placeholder="输入驳回原因（选填）"
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectOpen(false)}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleReject} disabled={submitting}>
              确认驳回
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
