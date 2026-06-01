import { useEffect, useMemo, useState } from "react"
import { Link } from "react-router-dom"
import { api } from "@/lib/api"
import type { TicketListItem, TicketStatus, APIKey, Program } from "@/types"
import { TICKET_STATUS_LABELS } from "@/types"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog"
import { toast } from "sonner"

const STATUS_COLORS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending_dryrun: "secondary",
  dryrun_failed: "outline",
  pending_approval: "default",
  approved: "secondary",
  executing: "secondary",
  done: "default",
  exec_failed: "destructive",
  rejected: "destructive",
}

const STATUS_SORT_ORDER: Record<TicketStatus, number> = {
  exec_failed: 0,
  dryrun_failed: 0,
  pending_approval: 1,
  executing: 2,
  approved: 2,
  done: 3,
  rejected: 4,
  pending_dryrun: 3,
}

function statusPriority(s: TicketStatus) {
  return STATUS_SORT_ORDER[s] ?? 5
}

function borderClass(s: TicketStatus) {
  switch (s) {
    case "exec_failed":
    case "dryrun_failed":
      return "border-l-4 border-l-red-500"
    case "pending_approval":
      return "border-l-4 border-l-yellow-500"
    case "done":
      return "border-l-4 border-l-green-500"
    default:
      return "border-l-4 border-l-gray-300"
  }
}

function truncateArgs(args: string[] | null | undefined, maxLen = 40) {
  if (!args || !Array.isArray(args)) return "-"
  const joined = args.join(" ")
  if (joined.length <= maxLen) return joined
  return joined.slice(0, maxLen) + "..."
}

export function TicketListPage() {
  const [allTickets, setAllTickets] = useState<TicketListItem[]>([])
  const [status, setStatus] = useState<TicketStatus | "">("")
  const [loading, setLoading] = useState(true)

  // 提交对话框状态
  const [submitOpen, setSubmitOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [submitError, setSubmitError] = useState("")
  const [apiKeys, setApiKeys] = useState<{ id: number; name: string }[]>([])
  const [programs, setPrograms] = useState<{ id: number; project: string; name: string }[]>([])
  const [form, setForm] = useState({ apiKeyId: "", programId: "", argsText: "" })

  useEffect(() => {
    setLoading(true)
    api.get<TicketListItem[]>("/tickets")
      .then((ts) => setAllTickets(ts))
      .catch(() => setAllTickets([]))
      .finally(() => setLoading(false))
  }, [])

  const counts = useMemo(() => {
    const c: Record<string, number> = {}
    for (const t of allTickets) {
      c[t.status] = (c[t.status] || 0) + 1
    }
    return c
  }, [allTickets])

  const sorted = useMemo(() => {
    return [...allTickets].sort((a, b) => {
      const pa = statusPriority(a.status)
      const pb = statusPriority(b.status)
      if (pa !== pb) return pa - pb
      return new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
    })
  }, [allTickets])

  const filtered = useMemo(() => {
    if (!status) return sorted
    return sorted.filter((t) => t.status === status)
  }, [sorted, status])

  const openSubmitDialog = () => {
    setSubmitError("")
    setForm({ apiKeyId: "", programId: "", argsText: "" })
    setSubmitOpen(true)
    Promise.all([
      api.get<APIKey[]>("/api-keys").catch(() => []),
      api.get<Program[]>("/programs").catch(() => []),
    ]).then(([keys, ps]) => {
      setApiKeys(keys)
      setPrograms(ps)
    })
  }

  const handleSubmit = async () => {
    if (!form.apiKeyId || !form.programId) {
      setSubmitError("请选择 API Key 和程序")
      return
    }
    const p = programs.find((x) => x.id === Number(form.programId))
    if (!p) return

    const args = form.argsText
      .split("\n")
      .map((s) => s.trim())
      .filter(Boolean)

    setSubmitting(true)
    setSubmitError("")
    try {
      await api.post("/tickets/submit", {
        api_key_id: Number(form.apiKeyId),
        project: p.project,
        name: p.name,
        args,
      })
      toast.success("工单提交成功")
      setSubmitOpen(false)
      // Refresh list
      setLoading(true)
      api.get<TicketListItem[]>("/tickets")
        .then((ts) => setAllTickets(ts))
        .catch(() => setAllTickets([]))
        .finally(() => setLoading(false))
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "提交失败")
    } finally {
      setSubmitting(false)
    }
  }

  const tabs: { label: string; value: TicketStatus | "" }[] = [
    { label: `全部 (${allTickets.length})`, value: "" },
    { label: `待审批 (${counts["pending_approval"] || 0})`, value: "pending_approval" },
    { label: `已完成 (${counts["done"] || 0})`, value: "done" },
    { label: `已驳回 (${counts["rejected"] || 0})`, value: "rejected" },
    { label: `执行失败 (${counts["exec_failed"] || 0})`, value: "exec_failed" },
    { label: `Dry-run 失败 (${counts["dryrun_failed"] || 0})`, value: "dryrun_failed" },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">工单列表</h1>
        <Button onClick={openSubmitDialog}>提交工单</Button>
      </div>

      <Tabs value={status} onValueChange={(v) => setStatus(v as TicketStatus | "")}>
        <TabsList>
          {tabs.map((f) => (
            <TabsTrigger key={f.value} value={f.value}>
              {f.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      {loading ? (
        <p className="text-muted-foreground text-sm">加载中...</p>
      ) : filtered.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center">暂无工单</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-48">程序</TableHead>
              <TableHead>参数预览</TableHead>
              <TableHead className="w-32">提交来源</TableHead>
              <TableHead className="w-28">状态</TableHead>
              <TableHead className="w-44">提交时间</TableHead>
              <TableHead className="w-20">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.map((t) => (
              <TableRow key={t.id} className={borderClass(t.status)}>
                <TableCell>
                  <div className="font-medium">{t.program_project || "-"}</div>
                  <div className="text-sm text-muted-foreground">{t.program_name || "-"}</div>
                </TableCell>
                <TableCell className="font-mono text-sm">
                  {truncateArgs(t.args)}
                </TableCell>
                <TableCell className="text-muted-foreground">
                  {t.submitted_by_name}
                </TableCell>
                <TableCell>
                  <Badge variant={STATUS_COLORS[t.status] || "default"}>
                    {TICKET_STATUS_LABELS[t.status]}
                  </Badge>
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {new Date(t.created_at).toLocaleString("zh-CN")}
                </TableCell>
                <TableCell>
                  <Link to={`/tickets/${t.id}`}>
                    <Button variant="ghost" size="sm">
                      查看
                    </Button>
                  </Link>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* 提交工单对话框 */}
      <Dialog open={submitOpen} onOpenChange={setSubmitOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>提交工单</DialogTitle>
            <DialogDescription>模拟 AI 提交工单，走完整的 dry-run → 审批流程</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1">
              <Label>API Key</Label>
              {apiKeys.length === 0 ? (
                <p className="text-sm text-muted-foreground">暂无可用 API Key，请先前往 API Key 管理页创建</p>
              ) : (
                <Select value={form.apiKeyId} onValueChange={(v) => { if (v) setForm({ ...form, apiKeyId: v }) }}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="选择 API Key" />
                  </SelectTrigger>
                  <SelectContent>
                    {apiKeys.map((k) => (
                      <SelectItem key={k.id} value={String(k.id)}>
                        {k.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
            <div className="space-y-1">
              <Label>程序</Label>
              {programs.length === 0 ? (
                <p className="text-sm text-muted-foreground">暂无已注册程序，请先前往程序管理页注册</p>
              ) : (
                <Select value={form.programId} onValueChange={(v) => { if (v) setForm({ ...form, programId: v }) }}>
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="选择程序" />
                  </SelectTrigger>
                  <SelectContent>
                    {programs.map((p) => (
                      <SelectItem key={p.id} value={String(p.id)}>
                        {p.project}/{p.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            </div>
            <div className="space-y-1">
              <Label>参数（每行一个）</Label>
              <Textarea
                placeholder={"--env\nprod"}
                rows={3}
                value={form.argsText}
                onChange={(e) => setForm({ ...form, argsText: e.target.value })}
              />
            </div>
            {submitError && (
              <p className="text-sm text-destructive">{submitError}</p>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setSubmitOpen(false)}>取消</Button>
            <Button onClick={handleSubmit} disabled={submitting || apiKeys.length === 0 || programs.length === 0}>
              {submitting ? "提交中..." : "提交"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
