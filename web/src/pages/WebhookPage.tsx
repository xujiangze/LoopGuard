import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import type { WebhookConfig, WebhookDelivery, Program, WebhookEventType } from "@/types"
import { WEBHOOK_EVENT_LABELS } from "@/types"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription } from "@/components/ui/dialog"
import { toast } from "sonner"

const ALL_EVENTS: WebhookEventType[] = [
  "ticket.pending_approval",
  "ticket.dryrun_failed",
  "ticket.done",
  "ticket.exec_failed",
  "ticket.rejected",
]

export function WebhookPage() {
  const [webhooks, setWebhooks] = useState<WebhookConfig[]>([])
  const [programs, setPrograms] = useState<Program[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<WebhookConfig | null>(null)
  const [deliveryTarget, setDeliveryTarget] = useState<WebhookConfig | null>(null)
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([])

  const [form, setForm] = useState({
    program_id: 0,
    name: "",
    url: "",
    event_types: [] as WebhookEventType[],
  })

  const fetchData = () => {
    setLoading(true)
    Promise.all([
      api.webhooks.list(),
      api.get<Program[]>("/programs").catch(() => []),
    ]).then(([ws, ps]) => {
      setWebhooks(ws ?? [])
      setPrograms(ps ?? [])
    }).finally(() => setLoading(false))
  }

  useEffect(() => { fetchData() }, [])

  const programName = (pid: number) => {
    const p = programs.find((p) => p.id === pid)
    return p ? `${p.project}/${p.name}` : `#${pid}`
  }

  const handleCreate = async () => {
    if (!form.program_id) { toast.error("请选择程序"); return }
    if (!form.name.trim()) { toast.error("请输入名称"); return }
    if (!form.url.trim()) { toast.error("请输入 Webhook URL"); return }
    if (!form.url.includes("qyapi.weixin.qq.com")) { toast.error("URL 必须包含 qyapi.weixin.qq.com"); return }
    if (form.event_types.length === 0) { toast.error("请至少选择一个事件类型"); return }
    try {
      await api.webhooks.create({
        program_id: form.program_id,
        name: form.name,
        url: form.url,
        enabled: true,
        event_types: form.event_types.join(","),
      })
      toast.success("Webhook 创建成功")
      setCreateOpen(false)
      setForm({ program_id: 0, name: "", url: "", event_types: [] })
      fetchData()
    } catch (err) {
      toast.error("创建失败", { description: err instanceof Error ? err.message : "" })
    }
  }

  const handleToggle = async (w: WebhookConfig) => {
    try {
      await api.webhooks.toggle(w.id, !w.enabled)
      fetchData()
    } catch { toast.error("操作失败") }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.webhooks.delete(deleteTarget.id)
      toast.success("已删除")
      setDeleteTarget(null)
      fetchData()
    } catch { toast.error("删除失败") }
  }

  const loadDeliveries = async (w: WebhookConfig) => {
    setDeliveryTarget(w)
    try {
      const ds = await api.webhooks.deliveries(w.id)
      setDeliveries(ds ?? [])
    } catch { setDeliveries([]) }
  }

  const toggleEventType = (evt: WebhookEventType) => {
    setForm((prev) => ({
      ...prev,
      event_types: prev.event_types.includes(evt)
        ? prev.event_types.filter((e) => e !== evt)
        : [...prev.event_types, evt],
    }))
  }

  if (loading) return <p className="text-muted-foreground text-sm">加载中...</p>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">Webhook 管理</h1>
        <Button onClick={() => setCreateOpen(true)}>创建 Webhook</Button>
      </div>

      {webhooks.length === 0 ? (
        <p className="text-muted-foreground text-sm">暂无 Webhook 配置</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>程序</TableHead>
              <TableHead>事件类型</TableHead>
              <TableHead className="w-20">状态</TableHead>
              <TableHead className="w-48 text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {webhooks.map((w) => (
              <TableRow key={w.id}>
                <TableCell className="font-medium">{w.name}</TableCell>
                <TableCell className="text-muted-foreground">{programName(w.program_id)}</TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-1">
                    {w.event_types.split(",").map((e) => (
                      <Badge key={e} variant="secondary" className="text-xs">
                        {WEBHOOK_EVENT_LABELS[e as WebhookEventType] ?? e}
                      </Badge>
                    ))}
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Switch checked={w.enabled} onCheckedChange={() => handleToggle(w)} />
                    <span className="text-xs text-muted-foreground">{w.enabled ? "启用" : "禁用"}</span>
                  </div>
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    <Button variant="outline" size="sm" onClick={() => loadDeliveries(w)}>
                      投递记录
                    </Button>
                    <Button variant="outline" size="sm" className="text-destructive" onClick={() => setDeleteTarget(w)}>
                      删除
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* 创建 Webhook 弹窗 */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>创建 Webhook</DialogTitle>
            <DialogDescription>配置企业微信 Webhook 通知</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>程序</Label>
              <select
                className="w-full mt-1 rounded-md border bg-background px-3 py-2 text-sm"
                value={form.program_id}
                onChange={(e) => setForm((prev) => ({ ...prev, program_id: Number(e.target.value) }))}
              >
                <option value={0}>选择程序...</option>
                {programs.map((p) => (
                  <option key={p.id} value={p.id}>{p.project}/{p.name}</option>
                ))}
              </select>
            </div>
            <div>
              <Label>名称</Label>
              <Input className="mt-1" value={form.name} onChange={(e) => setForm((prev) => ({ ...prev, name: e.target.value }))} placeholder="Webhook 名称" />
            </div>
            <div>
              <Label>Webhook URL</Label>
              <Input className="mt-1" value={form.url} onChange={(e) => setForm((prev) => ({ ...prev, url: e.target.value }))} placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." />
            </div>
            <div>
              <Label>事件类型</Label>
              <div className="mt-1 flex flex-wrap gap-2">
                {ALL_EVENTS.map((evt) => (
                  <label key={evt} className="flex items-center gap-1.5 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={form.event_types.includes(evt)}
                      onChange={() => toggleEventType(evt)}
                      className="rounded"
                    />
                    <span className="text-sm">{WEBHOOK_EVENT_LABELS[evt]}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>取消</Button>
            <Button onClick={handleCreate}>创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认弹窗 */}
      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>
              确定删除 Webhook <strong>{deleteTarget?.name}</strong>？删除后相关投递记录仍会保留。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 投递记录弹窗 */}
      <Dialog open={!!deliveryTarget} onOpenChange={(open) => !open && setDeliveryTarget(null)}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>投递记录 — {deliveryTarget?.name}</DialogTitle>
            <DialogDescription>Webhook 发送历史记录</DialogDescription>
          </DialogHeader>
          {deliveries.length === 0 ? (
            <p className="text-muted-foreground text-sm py-4">暂无投递记录</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>工单</TableHead>
                  <TableHead>事件</TableHead>
                  <TableHead className="w-20">状态码</TableHead>
                  <TableHead className="w-36">时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {deliveries.map((d) => (
                  <TableRow key={d.id}>
                    <TableCell className="text-muted-foreground">{d.id}</TableCell>
                    <TableCell>#{d.ticket_id}</TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="text-xs">
                        {WEBHOOK_EVENT_LABELS[d.event_type as WebhookEventType] ?? d.event_type}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={d.status_code >= 200 && d.status_code < 300 ? "default" : "destructive"}>
                        {d.status_code}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">{d.delivered_at ?? "-"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
