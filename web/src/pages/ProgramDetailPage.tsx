import { useEffect, useState } from "react"
import { useParams, useNavigate } from "react-router-dom"
import { api } from "@/lib/api"
import type { Program, ProgramVersion, ProgramFileInfo, User, WebhookConfig, WebhookDelivery, WebhookEventType } from "@/types"
import { WEBHOOK_EVENT_LABELS } from "@/types"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table"
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from "@/components/ui/dialog"
import {
  Collapsible, CollapsibleContent, CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { Switch } from "@/components/ui/switch"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"

export function ProgramDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [program, setProgram] = useState<Program | null>(null)
  const [users, setUsers] = useState<User[]>([])
  const [files, setFiles] = useState<ProgramFileInfo[]>([])
  const [versions, setVersions] = useState<ProgramVersion[]>([])
  const [loading, setLoading] = useState(true)
  const [fileContent, setFileContent] = useState<{ name: string; content: string } | null>(null)
  const [versionFiles, setVersionFiles] = useState<Record<number, ProgramFileInfo[]>>({})
  const [versionFileContent, setVersionFileContent] = useState<{ version: number; name: string; content: string } | null>(null)
  const [rollbackTarget, setRollbackTarget] = useState<ProgramVersion | null>(null)
  const [rollbacking, setRollbacking] = useState(false)
  const [webhooks, setWebhooks] = useState<WebhookConfig[]>([])
  const [webhookCreateOpen, setWebhookCreateOpen] = useState(false)
  const [webhookDeleteTarget, setWebhookDeleteTarget] = useState<WebhookConfig | null>(null)
  const [webhookDeliveryTarget, setWebhookDeliveryTarget] = useState<WebhookConfig | null>(null)
  const [webhookDeliveries, setWebhookDeliveries] = useState<WebhookDelivery[]>([])
  const [webhookForm, setWebhookForm] = useState({ name: "", url: "", event_types: [] as WebhookEventType[] })

  const ALL_WEBHOOK_EVENTS: WebhookEventType[] = [
    "ticket.pending_approval", "ticket.dryrun_failed", "ticket.done", "ticket.exec_failed", "ticket.rejected",
  ]

  const fetchProgram = () => {
    if (!id) return
    setLoading(true)
    Promise.all([
      api.get<Program>(`/programs/${id}`).catch(() => null),
      api.get<User[]>("/users").catch(() => []),
    ]).then(([p, us]) => {
      setProgram(p)
      setUsers(us)
      if (p) {
        api.get<ProgramFileInfo[]>(`/programs/${p.id}/files`).then(setFiles).catch(() => setFiles([]))
        api.get<ProgramVersion[]>(`/programs/${p.id}/versions`).then(setVersions).catch(() => setVersions([]))
        api.webhooks.list(p.id).then(setWebhooks).catch(() => setWebhooks([]))
      }
    }).finally(() => setLoading(false))
  }

  useEffect(() => { fetchProgram() }, [id])

  const userName = (uid: number) => users.find((u) => u.id === uid)?.username ?? String(uid)

  const loadFileContent = async (filename: string) => {
    if (!program) return
    if (fileContent?.name === filename) { setFileContent(null); return }
    try {
      const content = await api.getText(`/programs/${program.id}/files/${encodeURIComponent(filename)}`)
      setFileContent({ name: filename, content })
    } catch { toast.error("读取文件失败") }
  }

  const loadVersionFiles = async (v: ProgramVersion) => {
    if (versionFiles[v.version]) return
    try {
      const files = await api.get<ProgramFileInfo[]>(`/programs/${program!.id}/versions/${v.version}/files`)
      setVersionFiles((prev) => ({ ...prev, [v.version]: files }))
    } catch { toast.error("读取版本文件列表失败") }
  }

  const loadVersionFileContent = async (version: number, filename: string) => {
    if (versionFileContent?.version === version && versionFileContent?.name === filename) {
      setVersionFileContent(null); return
    }
    try {
      const content = await api.getText(`/programs/${program!.id}/versions/${version}/files/${encodeURIComponent(filename)}`)
      setVersionFileContent({ version, name: filename, content })
    } catch { toast.error("读取文件失败") }
  }

  const handleRollback = async () => {
    if (!rollbackTarget || !program) return
    setRollbacking(true)
    try {
      await api.post(`/programs/${program.id}/rollback`, { version: rollbackTarget.version })
      toast.success(`已回滚到 v${rollbackTarget.version}`)
      setRollbackTarget(null)
      fetchProgram()
    } catch (err) {
      toast.error("回滚失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setRollbacking(false)
    }
  }

  if (loading) return <p className="text-muted-foreground text-sm">加载中...</p>
  if (!program) return <p className="text-muted-foreground text-sm">程序不存在</p>

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="sm" onClick={() => navigate("/admin/programs")}>← 返回</Button>
        <h1 className="text-2xl font-semibold">{program.project}/{program.name}</h1>
        <Badge variant={program.enabled ? "default" : "outline"}>{program.enabled ? "启用" : "禁用"}</Badge>
      </div>

      <Tabs defaultValue="info">
        <TabsList>
          <TabsTrigger value="info">基本信息</TabsTrigger>
          <TabsTrigger value="files">文件</TabsTrigger>
          <TabsTrigger value="versions">版本历史</TabsTrigger>
          <TabsTrigger value="webhooks">Webhook</TabsTrigger>
        </TabsList>

        <TabsContent value="info">
          <Card>
            <CardHeader><CardTitle>程序信息</CardTitle></CardHeader>
            <CardContent>
              <dl className="grid grid-cols-2 gap-4 text-sm">
                <div><dt className="text-muted-foreground">项目</dt><dd className="font-mono mt-1">{program.project}</dd></div>
                <div><dt className="text-muted-foreground">程序名</dt><dd className="font-mono mt-1">{program.name}</dd></div>
                <div><dt className="text-muted-foreground">解释器</dt><dd className="font-mono mt-1">{program.interpreter}</dd></div>
                <div><dt className="text-muted-foreground">入口文件</dt><dd className="font-mono mt-1">{program.entry_file}</dd></div>
                <div><dt className="text-muted-foreground">审批人</dt><dd className="mt-1">{userName(program.approver_id)}</dd></div>
                <div><dt className="text-muted-foreground">超时</dt><dd className="mt-1">{program.timeout_sec}s</dd></div>
                <div><dt className="text-muted-foreground">当前版本</dt><dd className="mt-1">v{program.current_version}</dd></div>
                <div><dt className="text-muted-foreground">状态</dt><dd className="mt-1"><Badge variant={program.enabled ? "default" : "outline"}>{program.enabled ? "启用" : "禁用"}</Badge></dd></div>
                {program.help_text && (
                  <div className="col-span-2"><dt className="text-muted-foreground">Help 输出</dt><dd className="mt-1 whitespace-pre-wrap text-xs bg-muted/50 rounded p-3 font-mono max-h-48 overflow-y-auto">{program.help_text}</dd></div>
                )}
              </dl>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="files">
          <Card>
            <CardHeader><CardTitle>当前文件（v{program.current_version}）</CardTitle></CardHeader>
            <CardContent>
              {files.length === 0 ? (
                <p className="text-muted-foreground text-sm">无文件</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>文件名</TableHead>
                      <TableHead className="w-24">大小</TableHead>
                      <TableHead className="w-20">入口</TableHead>
                      <TableHead className="w-40">修改时间</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {files.map((f) => (
                      <Collapsible key={f.name} open={fileContent?.name === f.name} onOpenChange={() => loadFileContent(f.name)}>
                        <TableRow className="cursor-pointer" onClick={() => loadFileContent(f.name)}>
                          <TableCell className="font-mono text-primary">{f.name}</TableCell>
                          <TableCell className="text-muted-foreground">{formatSize(f.size)}</TableCell>
                          <TableCell>{f.is_entry && <Badge variant="secondary">入口</Badge>}</TableCell>
                          <TableCell className="text-muted-foreground text-xs">{f.mod_time}</TableCell>
                        </TableRow>
                        <CollapsibleContent>
                          <tr><td colSpan={4} className="bg-muted/30 p-3">
                            <pre className="text-xs font-mono whitespace-pre-wrap max-h-64 overflow-y-auto">{fileContent?.name === f.name ? fileContent.content : "加载中..."}</pre>
                          </td></tr>
                        </CollapsibleContent>
                      </Collapsible>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="versions">
          <Card>
            <CardHeader><CardTitle>版本历史</CardTitle></CardHeader>
            <CardContent className="space-y-3">
              {versions.length === 0 ? (
                <p className="text-muted-foreground text-sm">无版本记录</p>
              ) : versions.map((v) => (
                <Collapsible key={v.id} onOpenChange={(open) => open && loadVersionFiles(v)}>
                  <div className="flex items-center justify-between p-3 bg-muted/30 rounded-lg">
                    <CollapsibleTrigger className="flex items-center gap-3 cursor-pointer text-left">
                      <span className="font-mono font-semibold">v{v.version}</span>
                      {v.is_rollback && <Badge variant="outline" className="text-xs">回滚</Badge>}
                      <span className="text-xs text-muted-foreground">{v.created_at}</span>
                      {v.created_by && <span className="text-xs text-muted-foreground">by {v.created_by}</span>}
                    </CollapsibleTrigger>
                    {v.version !== program.current_version && (
                      <Button variant="outline" size="sm" onClick={() => setRollbackTarget(v)}>
                        回滚到此版本
                      </Button>
                    )}
                  </div>
                  <CollapsibleContent>
                    <div className="ml-4 mt-2 space-y-1">
                      {(versionFiles[v.version] || []).map((f) => (
                        <div key={f.name} className="flex items-center gap-2">
                          <span
                            className="font-mono text-sm text-primary cursor-pointer hover:underline"
                            onClick={() => loadVersionFileContent(v.version, f.name)}
                          >
                            {f.name}
                          </span>
                          <span className="text-xs text-muted-foreground">{formatSize(f.size)}</span>
                          {f.is_entry && <Badge variant="secondary" className="text-xs">入口</Badge>}
                        </div>
                      ))}
                      {versionFileContent?.version === v.version && (
                        <pre className="text-xs font-mono bg-muted/50 p-3 rounded mt-2 whitespace-pre-wrap max-h-48 overflow-y-auto">
                          {versionFileContent.content}
                        </pre>
                      )}
                    </div>
                  </CollapsibleContent>
                </Collapsible>
              ))}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="webhooks">
          <Card>
            <CardHeader className="flex-row items-center justify-between">
              <CardTitle>Webhook 配置</CardTitle>
              <Button size="sm" onClick={() => setWebhookCreateOpen(true)}>创建 Webhook</Button>
            </CardHeader>
            <CardContent>
              {webhooks.length === 0 ? (
                <p className="text-muted-foreground text-sm">暂无 Webhook，点击"创建 Webhook"配置企业微信通知</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>名称</TableHead>
                      <TableHead>事件类型</TableHead>
                      <TableHead className="w-20">状态</TableHead>
                      <TableHead className="w-48 text-right">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {webhooks.map((w) => (
                      <TableRow key={w.id}>
                        <TableCell className="font-medium">{w.name}</TableCell>
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
                            <Switch checked={w.enabled} onCheckedChange={async () => {
                              try { await api.webhooks.toggle(w.id, !w.enabled); api.webhooks.list(program!.id).then(setWebhooks) }
                              catch { toast.error("操作失败") }
                            }} />
                            <span className="text-xs text-muted-foreground">{w.enabled ? "启用" : "禁用"}</span>
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-2">
                            <Button variant="outline" size="sm" onClick={async () => {
                              setWebhookDeliveryTarget(w)
                              try { const ds = await api.webhooks.deliveries(w.id); setWebhookDeliveries(ds ?? []) } catch { setWebhookDeliveries([]) }
                            }}>投递记录</Button>
                            <Button variant="outline" size="sm" className="text-destructive" onClick={() => setWebhookDeleteTarget(w)}>删除</Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Webhook 标签页内容 - 在 Tabs 外独立渲染弹窗 */}
      {/* 创建 Webhook 弹窗 */}
      <Dialog open={webhookCreateOpen} onOpenChange={setWebhookCreateOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>创建 Webhook</DialogTitle>
            <DialogDescription>为 {program?.project}/{program?.name} 配置企业微信通知</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <Label>名称</Label>
              <Input className="mt-1" value={webhookForm.name} onChange={(e) => setWebhookForm((p) => ({ ...p, name: e.target.value }))} placeholder="Webhook 名称" />
            </div>
            <div>
              <Label>Webhook URL</Label>
              <Input className="mt-1" value={webhookForm.url} onChange={(e) => setWebhookForm((p) => ({ ...p, url: e.target.value }))} placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." />
            </div>
            <div>
              <Label>事件类型</Label>
              <div className="mt-1 flex flex-wrap gap-2">
                {ALL_WEBHOOK_EVENTS.map((evt) => (
                  <label key={evt} className="flex items-center gap-1.5 cursor-pointer">
                    <input type="checkbox" checked={webhookForm.event_types.includes(evt)} onChange={() => setWebhookForm((p) => ({ ...p, event_types: p.event_types.includes(evt) ? p.event_types.filter((e) => e !== evt) : [...p.event_types, evt] }))} className="rounded" />
                    <span className="text-sm">{WEBHOOK_EVENT_LABELS[evt]}</span>
                  </label>
                ))}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setWebhookCreateOpen(false)}>取消</Button>
            <Button onClick={async () => {
              if (!program) return
              if (!webhookForm.name.trim()) { toast.error("请输入名称"); return }
              if (!webhookForm.url.includes("qyapi.weixin.qq.com")) { toast.error("URL 必须包含 qyapi.weixin.qq.com"); return }
              if (webhookForm.event_types.length === 0) { toast.error("请选择事件类型"); return }
              try {
                await api.webhooks.create({ program_id: program.id, name: webhookForm.name, url: webhookForm.url, enabled: true, event_types: webhookForm.event_types.join(",") })
                toast.success("Webhook 创建成功")
                setWebhookCreateOpen(false)
                setWebhookForm({ name: "", url: "", event_types: [] })
                api.webhooks.list(program.id).then(setWebhooks)
              } catch (err) { toast.error("创建失败", { description: err instanceof Error ? err.message : "" }) }
            }}>创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除 Webhook 弹窗 */}
      <Dialog open={!!webhookDeleteTarget} onOpenChange={(open) => !open && setWebhookDeleteTarget(null)}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
            <DialogDescription>确定删除 Webhook <strong>{webhookDeleteTarget?.name}</strong>？</DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setWebhookDeleteTarget(null)}>取消</Button>
            <Button variant="destructive" onClick={async () => {
              if (!webhookDeleteTarget || !program) return
              try {
                await api.webhooks.delete(webhookDeleteTarget.id)
                toast.success("已删除")
                setWebhookDeleteTarget(null)
                api.webhooks.list(program.id).then(setWebhooks)
              } catch { toast.error("删除失败") }
            }}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Webhook 投递记录弹窗 */}
      <Dialog open={!!webhookDeliveryTarget} onOpenChange={(open) => !open && setWebhookDeliveryTarget(null)}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>投递记录 — {webhookDeliveryTarget?.name}</DialogTitle>
          </DialogHeader>
          {webhookDeliveries.length === 0 ? (
            <p className="text-muted-foreground text-sm py-4">暂无投递记录</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>工单</TableHead>
                  <TableHead>事件</TableHead>
                  <TableHead>状态码</TableHead>
                  <TableHead>时间</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {webhookDeliveries.map((d) => (
                  <TableRow key={d.id}>
                    <TableCell>#{d.ticket_id}</TableCell>
                    <TableCell><Badge variant="secondary" className="text-xs">{WEBHOOK_EVENT_LABELS[d.event_type as WebhookEventType] ?? d.event_type}</Badge></TableCell>
                    <TableCell><Badge variant={d.status_code >= 200 && d.status_code < 300 ? "default" : "destructive"}>{d.status_code}</Badge></TableCell>
                    <TableCell className="text-xs text-muted-foreground">{d.delivered_at ?? "-"}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </DialogContent>
      </Dialog>

      {/* 回滚确认弹窗 */}
      <Dialog open={!!rollbackTarget} onOpenChange={(open) => !open && setRollbackTarget(null)}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>确认回滚</DialogTitle>
            <DialogDescription>
              确定回滚到 <strong>v{rollbackTarget?.version}</strong>？将创建新版本（v{program.current_version + 1}），标记为回滚版本。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRollbackTarget(null)}>取消</Button>
            <Button onClick={handleRollback} disabled={rollbacking}>
              {rollbacking ? "回滚中..." : "确认回滚"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}
