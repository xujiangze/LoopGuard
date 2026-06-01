import { useEffect, useRef, useState } from "react"
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
    entry_file: "",
    interpreter: "",
    approver_id: "",
    timeout_sec: "300",
  })
  const [createFiles, setCreateFiles] = useState<FileList | null>(null)

  const [editForm, setEditForm] = useState({
    enabled: true,
    interpreter: "",
    approver_id: "",
    timeout_sec: "300",
  })
  const [editFiles, setEditFiles] = useState<FileList | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const editFileInputRef = useRef<HTMLInputElement>(null)

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
    if (!form.project || !form.name || !form.entry_file || !form.interpreter || !form.approver_id) {
      toast.error("请填写所有必填字段")
      return
    }
    if (!createFiles || createFiles.length === 0) {
      toast.error("请上传文件")
      return
    }

    const fd = new FormData()
    fd.append("project", form.project)
    fd.append("name", form.name)
    fd.append("entry_file", form.entry_file)
    fd.append("interpreter", form.interpreter)
    fd.append("approver_id", form.approver_id)
    fd.append("timeout_sec", form.timeout_sec)
    for (let i = 0; i < createFiles.length; i++) {
      fd.append("files", createFiles[i])
    }

    setSubmitting(true)
    try {
      await api.upload("/programs", fd)
      toast.success("程序注册成功", { description: `${form.project}/${form.name}` })
      setCreateOpen(false)
      setForm({ project: "", name: "", entry_file: "", interpreter: "", approver_id: "", timeout_sec: "300" })
      setCreateFiles(null)
      if (fileInputRef.current) fileInputRef.current.value = ""
      fetchData()
    } catch (err) {
      toast.error("创建失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setSubmitting(false)
    }
  }

  const openEdit = (p: Program) => {
    setEditTarget(p)
    setEditForm({ enabled: p.enabled, interpreter: p.interpreter, approver_id: String(p.approver_id), timeout_sec: String(p.timeout_sec) })
    setEditFiles(null)
    if (editFileInputRef.current) editFileInputRef.current.value = ""
    setEditOpen(true)
  }

  const handleEdit = async () => {
    if (!editTarget) return
    setSubmitting(true)
    try {
      const fd = new FormData()
      fd.append("enabled", String(editForm.enabled))
      if (editForm.interpreter) fd.append("interpreter", editForm.interpreter)
      if (editForm.approver_id) fd.append("approver_id", editForm.approver_id)
      if (editForm.timeout_sec) fd.append("timeout_sec", editForm.timeout_sec)
      if (editFiles) {
        for (let i = 0; i < editFiles.length; i++) {
          fd.append("files", editFiles[i])
        }
      }
      await api.uploadPut(`/programs/${editTarget.id}`, fd)
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
      const fd = new FormData()
      fd.append("enabled", String(!p.enabled))
      await api.uploadPut(`/programs/${p.id}`, fd)
      toast.success(p.enabled ? "已禁用" : "已启用", { description: `${p.project}/${p.name}` })
      fetchData()
    } catch (err) {
      toast.error("操作失败", { description: err instanceof Error ? err.message : "未知错误" })
    }
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
              <TableHead>入口文件</TableHead>
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
                  <div className="text-xs text-muted-foreground mt-0.5">{p.interpreter}</div>
                </TableCell>
                <TableCell className="font-mono text-sm">{p.entry_file}</TableCell>
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
            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">基本信息</p>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1">
                  <Label>项目名 <span className="text-destructive">*</span></Label>
                  <Input placeholder="如 tsunami_ipban" value={form.project} onChange={(e) => setForm({ ...form, project: e.target.value })} />
                </div>
                <div className="space-y-1">
                  <Label>程序名 <span className="text-destructive">*</span></Label>
                  <Input placeholder="如 entry_ipban" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
                </div>
              </div>
              <div className="space-y-1">
                <Label>入口文件名 <span className="text-destructive">*</span></Label>
                <Input placeholder="如 entry_ipban.py" value={form.entry_file} onChange={(e) => setForm({ ...form, entry_file: e.target.value })} />
                <p className="text-xs text-muted-foreground">必须与上传文件中的文件名一致</p>
              </div>
              <div className="space-y-1">
                <Label>解释器 <span className="text-destructive">*</span></Label>
                <Input placeholder="如 python3、bash、node" value={form.interpreter} onChange={(e) => setForm({ ...form, interpreter: e.target.value })} />
              </div>
            </div>

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
              </div>
              <div className="space-y-1">
                <Label>超时时间</Label>
                <div className="flex items-center gap-2">
                  <Input type="number" className="w-24" value={form.timeout_sec} onChange={(e) => setForm({ ...form, timeout_sec: e.target.value })} />
                  <span className="text-sm text-muted-foreground">秒</span>
                </div>
              </div>
            </div>

            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">上传文件 <span className="text-destructive">*</span></p>
              <Input
                ref={fileInputRef}
                type="file"
                multiple
                onChange={(e) => setCreateFiles(e.target.files)}
              />
              {createFiles && (
                <p className="text-xs text-muted-foreground">
                  已选择 {createFiles.length} 个文件: {Array.from(createFiles).map((f) => f.name).join(", ")}
                </p>
              )}
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
              <Label>解释器</Label>
              <Input placeholder="如 python3" value={editForm.interpreter} onChange={(e) => setEditForm({ ...editForm, interpreter: e.target.value })} />
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
            <div className="space-y-3 p-3 bg-muted/50 rounded-lg">
              <p className="text-xs text-muted-foreground font-medium uppercase tracking-wide">更新文件（可选）</p>
              <Input
                ref={editFileInputRef}
                type="file"
                multiple
                onChange={(e) => setEditFiles(e.target.files)}
              />
              {editFiles && (
                <p className="text-xs text-muted-foreground">
                  已选择 {editFiles.length} 个文件，上传后将替换全部现有文件
                </p>
              )}
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
