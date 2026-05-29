import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import type { Program } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"

export function ProgramPage() {
  const [programs, setPrograms] = useState<Program[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [editOpen, setEditOpen] = useState(false)
  const [editTarget, setEditTarget] = useState<Program | null>(null)
  const [error, setError] = useState("")

  // Create form
  const [form, setForm] = useState({
    project: "",
    name: "",
    binary_path: "",
    approver_id: "",
    timeout_sec: "300",
    params_schema: "",
  })

  // Edit form
  const [editForm, setEditForm] = useState({
    enabled: true,
    approver_id: "",
    timeout_sec: "300",
  })

  const fetchPrograms = () => {
    setLoading(true)
    api
      .get<Program[]>("/programs")
      .then(setPrograms)
      .catch(() => setPrograms([]))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchPrograms()
  }, [])

  const handleCreate = async () => {
    setError("")
    try {
      await api.post("/programs", {
        project: form.project,
        name: form.name,
        binary_path: form.binary_path,
        approver_id: Number(form.approver_id),
        timeout_sec: Number(form.timeout_sec),
        params_schema: form.params_schema ? JSON.parse(form.params_schema) : null,
      })
      setCreateOpen(false)
      setForm({ project: "", name: "", binary_path: "", approver_id: "", timeout_sec: "300", params_schema: "" })
      fetchPrograms()
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建失败")
    }
  }

  const openEdit = (p: Program) => {
    setEditTarget(p)
    setEditForm({
      enabled: p.enabled,
      approver_id: String(p.approver_id),
      timeout_sec: String(p.timeout_sec),
    })
    setEditOpen(true)
  }

  const handleEdit = async () => {
    if (!editTarget) return
    setError("")
    try {
      await api.put(`/programs/${editTarget.id}`, {
        enabled: editForm.enabled,
        approver_id: Number(editForm.approver_id),
        timeout_sec: Number(editForm.timeout_sec),
      })
      setEditOpen(false)
      fetchPrograms()
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新失败")
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">程序管理</h1>
        <Button onClick={() => setCreateOpen(true)}>注册新程序</Button>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}

      {loading ? (
        <p className="text-muted-foreground text-sm">加载中...</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>项目/名称</TableHead>
              <TableHead>二进制路径</TableHead>
              <TableHead className="w-24">审批人 ID</TableHead>
              <TableHead className="w-20">状态</TableHead>
              <TableHead className="w-20">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {programs.map((p) => (
              <TableRow key={p.id}>
                <TableCell className="font-mono">
                  {p.project}/{p.name}
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">{p.binary_path}</TableCell>
                <TableCell>{p.approver_id}</TableCell>
                <TableCell>
                  <Badge variant={p.enabled ? "default" : "outline"}>
                    {p.enabled ? "启用" : "禁用"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Button variant="ghost" size="sm" onClick={() => openEdit(p)}>
                    编辑
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* 创建弹窗 */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>注册新程序</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <Label>项目名</Label>
              <Input
                value={form.project}
                onChange={(e) => setForm({ ...form, project: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>程序名</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>二进制路径</Label>
              <Input
                value={form.binary_path}
                onChange={(e) => setForm({ ...form, binary_path: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>审批人 ID</Label>
              <Input
                type="number"
                value={form.approver_id}
                onChange={(e) => setForm({ ...form, approver_id: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>超时（秒）</Label>
              <Input
                type="number"
                value={form.timeout_sec}
                onChange={(e) => setForm({ ...form, timeout_sec: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>参数白名单（JSON）</Label>
              <Input
                value={form.params_schema}
                onChange={(e) => setForm({ ...form, params_schema: e.target.value })}
                placeholder='{"key": "desc"}'
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              取消
            </Button>
            <Button onClick={handleCreate}>注册</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 编辑弹窗 */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>编辑程序</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <Label>启用</Label>
              <input
                type="checkbox"
                checked={editForm.enabled}
                onChange={(e) => setEditForm({ ...editForm, enabled: e.target.checked })}
              />
            </div>
            <div className="space-y-1">
              <Label>审批人 ID</Label>
              <Input
                type="number"
                value={editForm.approver_id}
                onChange={(e) => setEditForm({ ...editForm, approver_id: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>超时（秒）</Label>
              <Input
                type="number"
                value={editForm.timeout_sec}
                onChange={(e) => setEditForm({ ...editForm, timeout_sec: e.target.value })}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditOpen(false)}>
              取消
            </Button>
            <Button onClick={handleEdit}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
