import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import type { Program, User } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
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
} from "@/components/ui/dialog"
import { toast } from "sonner"

interface ParamEntry {
  key: string
  desc: string
}

export function ProgramPage() {
  const [programs, setPrograms] = useState<Program[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Program | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const [form, setForm] = useState({
    project: "",
    name: "",
    binary_path: "",
    approver_id: "",
    timeout_sec: "300",
  })
  const [params, setParams] = useState<ParamEntry[]>([{ key: "", desc: "" }])

  const [editForm, setEditForm] = useState({
    enabled: true,
    approver_id: "",
    timeout_sec: "300",
  })

  const fetchData = () => {
    setLoading(true)
    Promise.all([api.get<Program[]>("/programs"), api.get<User[]>("/users")])
      .then(([ps, us]) => {
        setPrograms(ps)
        setUsers(us)
      })
      .catch(() => {
        setPrograms([])
        setUsers([])
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => { fetchData() }, [])

  const userName = (id: number) => users.find((u) => u.id === id)?.username ?? String(id)
  const userInitial = (id: number) => {
    const name = userName(id)
    return name === String(id) ? "?" : name.charAt(0).toUpperCase()
  }

  const handleCreate = async () => {
    if (!form.project || !form.name || !form.binary_path || !form.approver_id) {
      toast.error("请填写所有必填字段")
      return
    }
    const schema: Record<string, string> = {}
    for (const p of params) {
      if (p.key.trim()) schema[p.key.trim()] = p.desc.trim()
    }
    setSubmitting(true)
    try {
      await api.post("/programs", {
        project: form.project,
        name: form.name,
        binary_path: form.binary_path,
        approver_id: Number(form.approver_id),
        timeout_sec: Number(form.timeout_sec) || 300,
        params_schema: Object.keys(schema).length > 0 ? schema : null,
      })
      toast.success("程序注册成功", { description: `${form.project}/${form.name} 已添加到白名单` })
      setCreateOpen(false)
      setForm({ project: "", name: "", binary_path: "", approver_id: "", timeout_sec: "300" })
      setParams([{ key: "", desc: "" }])
      fetchData()
    } catch (err) {
      toast.error("创建失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setSubmitting(false)
    }
  }

  const openEdit = (p: Program) => {
    setEditTarget(p)
    setEditForm({ enabled: p.enabled, approver_id: String(p.approver_id), timeout_sec: String(p.timeout_sec) })
    setEditOpen(true)
  }

  const handleEdit = async () => {
    if (!editTarget) return
    setSubmitting(true)
    try {
      await api.put(`/programs/${editTarget.id}`, {
        enabled: editForm.enabled,
        approver_id: Number(editForm.approver_id),
        timeout_sec: Number(editForm.timeout_sec) || 300,
      })
      toast.success("程序更新成功")
      setEditOpen(false)
      fetchData()
    } catch (err) {
      toast.error("更新失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setSubmitting(false)
    }
  }

  const toggleEnabled = async (p: Program) => {
    try {
      await api.put(`/programs/${p.id}`, { enabled: !p.enabled })
      toast.success(p.enabled ? "已禁用" : "已启用", { description: `${p.project}/${p.name}` })
      fetchData()
    } catch (err) {
      toast.error("操作失败", { description: err instanceof Error ? err.message : "未知错误" })
    }
  }

  const addParam = () => setParams([...params, { key: "", desc: "" }])
  const removeParam = (idx: number) => setParams(params.filter((_, i) => i !== idx))
  const updateParam = (idx: number, field: "key" | "desc", value: string) => {
    const next = [...params]
    next[idx][field] = value
    setParams(next)
  }

  if (loading) return <p className="text-muted-foreground text-sm">加载中...</p>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">程序管理</h1>
        <Button onClick={() => setCreateOpen(true)}>注册新程序</Button>
      </div>

      {programs.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center">暂无注册程序</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>项目 / 程序名</TableHead>
              <TableHead>审批人</TableHead>
              <TableHead className="w-20">超时</TableHead>
              <TableHead className="w-20">状态</TableHead>
              <TableHead className="w-28">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {programs.map((p) => (
              <TableRow key={p.id}>
                <TableCell>
                  <div className="font-mono font-medium">{p.project}/{p.name}</div>
                  <div className="text-xs text-muted-foreground mt-0.5">{p.binary_path}</div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <div className="w-6 h-6 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-semibold">
                      {userInitial(p.approver_id)}
                    </div>
                    <span className="text-sm">{userName(p.approver_id)}</span>
                  </div>
                </TableCell>
                <TableCell className="text-muted-foreground">{p.timeout_sec}s</TableCell>
                <TableCell>
                  <Badge variant={p.enabled ? "default" : "outline"}>
                    {p.enabled ? "启用" : "禁用"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-2">
                    <Button variant="ghost" size="sm" onClick={() => openEdit(p)}>编辑</Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className={p.enabled ? "text-destructive" : "text-green-600"}
                      onClick={() => toggleEnabled(p)}
                    >
                      {p.enabled ? "禁用" : "启用"}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* 创建弹窗 */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>注册新程序</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 max-h-[60vh] overflow-y-auto px-1">
            {/* 基本信息 */}
            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">基本信息</p>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label>项目名 <span className="text-destructive">*</span></Label>
                  <Input value={form.project} onChange={(e) => setForm({ ...form, project: e.target.value })} />
                </div>
                <div className="space-y-1">
                  <Label>程序名 <span className="text-destructive">*</span></Label>
                  <Input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
                </div>
              </div>
              <div className="space-y-1">
                <Label>二进制路径 <span className="text-destructive">*</span></Label>
                <Input value={form.binary_path} onChange={(e) => setForm({ ...form, binary_path: e.target.value })} />
              </div>
            </div>

            {/* 审批配置 */}
            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">审批配置</p>
              <div className="space-y-1">
                <Label>审批人 <span className="text-destructive">*</span></Label>
                <Select value={form.approver_id} onValueChange={(v) => { if (v) setForm({ ...form, approver_id: v }) }}>
                  <SelectTrigger><SelectValue placeholder="选择审批人" /></SelectTrigger>
                  <SelectContent>
                    {users.map((u) => (
                      <SelectItem key={u.id} value={String(u.id)}>
                        {u.username}（{u.role}）
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground">此程序的执行工单将自动分配给该审批人</p>
              </div>
              <div className="space-y-1">
                <Label>超时时间</Label>
                <div className="flex items-center gap-2">
                  <Input type="number" className="w-24" value={form.timeout_sec} onChange={(e) => setForm({ ...form, timeout_sec: e.target.value })} />
                  <span className="text-sm text-muted-foreground">秒</span>
                </div>
              </div>
            </div>

            {/* 参数白名单 */}
            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <div className="flex justify-between items-center">
                <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">参数白名单</p>
                <span className="text-xs text-muted-foreground">可选</span>
              </div>
              <p className="text-xs text-muted-foreground">定义 AI 提交工单时可传的参数名称，不在白名单中的参数将被拒绝</p>
              {params.map((p, idx) => (
                <div key={idx} className="flex items-center gap-2">
                  <Input
                    className="flex-1"
                    placeholder="参数名"
                    value={p.key}
                    onChange={(e) => updateParam(idx, "key", e.target.value)}
                  />
                  <Input
                    className="flex-[2]"
                    placeholder="说明"
                    value={p.desc}
                    onChange={(e) => updateParam(idx, "desc", e.target.value)}
                  />
                  <Button variant="ghost" size="sm" className="text-destructive px-2" onClick={() => removeParam(idx)} disabled={params.length <= 1}>
                    ✕
                  </Button>
                </div>
              ))}
              <Button variant="outline" size="sm" className="w-full border-dashed" onClick={addParam}>+ 添加参数</Button>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>取消</Button>
            <Button onClick={handleCreate} disabled={submitting}>
              {submitting ? "注册中..." : "注册"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 编辑弹窗 */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>编辑程序</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-center justify-between p-3 bg-muted/50 rounded-lg">
              <div>
                <Label>启用状态</Label>
                <p className="text-xs text-muted-foreground mt-0.5">禁用后 AI 将无法提交此程序的工单</p>
              </div>
              <Switch
                checked={editForm.enabled}
                onCheckedChange={(checked: boolean) => setEditForm({ ...editForm, enabled: checked })}
              />
            </div>
            <div className="space-y-1">
              <Label>审批人</Label>
              <Select value={editForm.approver_id} onValueChange={(v) => { if (v) setEditForm({ ...editForm, approver_id: v }) }}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  {users.map((u) => (
                    <SelectItem key={u.id} value={String(u.id)}>
                      {u.username}（{u.role}）
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1">
              <Label>超时（秒）</Label>
              <Input type="number" value={editForm.timeout_sec} onChange={(e) => setEditForm({ ...editForm, timeout_sec: e.target.value })} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditOpen(false)}>取消</Button>
            <Button onClick={handleEdit} disabled={submitting}>
              {submitting ? "保存中..." : "保存"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
