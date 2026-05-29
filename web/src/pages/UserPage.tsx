import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import type { User } from "@/types"
import { useAuth } from "@/hooks/useAuth"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import { toast } from "sonner"

export function UserPage() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [resetOpen, setResetOpen] = useState(false)
  const [resetTarget, setResetTarget] = useState<User | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const { userId } = useAuth()

  const [form, setForm] = useState({ username: "", password: "", confirmPwd: "", role: "user" })
  const [resetForm, setResetForm] = useState({ password: "", confirmPwd: "" })

  const fetchUsers = () => {
    setLoading(true)
    api
      .get<User[]>("/users")
      .then(setUsers)
      .catch(() => setUsers([]))
      .finally(() => setLoading(false))
  }

  useEffect(() => { fetchUsers() }, [])

  const handleCreate = async () => {
    if (!form.username || !form.password) {
      toast.error("请填写用户名和密码")
      return
    }
    if (form.password.length < 6) {
      toast.error("密码不能少于 6 位")
      return
    }
    if (form.password !== form.confirmPwd) {
      toast.error("两次密码输入不一致")
      return
    }
    setSubmitting(true)
    try {
      await api.post("/users", { username: form.username, password: form.password, role: form.role })
      toast.success("用户创建成功", { description: form.username })
      setCreateOpen(false)
      setForm({ username: "", password: "", confirmPwd: "", role: "user" })
      fetchUsers()
    } catch (err) {
      toast.error("创建失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setSubmitting(false)
    }
  }

  const handleReset = async () => {
    if (!resetTarget) return
    if (resetForm.password.length < 6) {
      toast.error("密码不能少于 6 位")
      return
    }
    if (resetForm.password !== resetForm.confirmPwd) {
      toast.error("两次密码输入不一致")
      return
    }
    setSubmitting(true)
    try {
      await api.put(`/users/${resetTarget.id}/password`, { password: resetForm.password })
      toast.success("密码已重置", { description: resetTarget.username })
      setResetOpen(false)
      setResetForm({ password: "", confirmPwd: "" })
    } catch (err) {
      toast.error("重置失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) return <p className="text-muted-foreground text-sm">加载中...</p>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">用户管理</h1>
        <Button onClick={() => setCreateOpen(true)}>创建用户</Button>
      </div>

      {users.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center">暂无用户数据</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>用户</TableHead>
              <TableHead className="w-24">角色</TableHead>
              <TableHead className="w-44">创建时间</TableHead>
              <TableHead className="w-28">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.map((u) => {
              const isSelf = u.id === userId
              const initial = u.username.charAt(0).toUpperCase()
              return (
                <TableRow key={u.id} className={isSelf ? "bg-amber-50 dark:bg-amber-950/20" : undefined}>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <div className="w-7 h-7 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-semibold">
                        {initial}
                      </div>
                      <div>
                        <span className="font-medium">{u.username}</span>
                        {isSelf && <span className="text-xs text-muted-foreground ml-2">当前登录</span>}
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={u.role === "admin" ? "default" : "secondary"}>
                      {u.role}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    {new Date(u.created_at).toLocaleString("zh-CN")}
                  </TableCell>
                  <TableCell>
                    {isSelf ? (
                      <span className="text-muted-foreground text-sm">—</span>
                    ) : (
                      <Button variant="ghost" size="sm" onClick={() => { setResetTarget(u); setResetOpen(true) }}>
                        重置密码
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      )}

      {/* 创建用户弹窗 */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>创建用户</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1">
                <Label>用户名 <span className="text-destructive">*</span></Label>
                <Input value={form.username} onChange={(e) => setForm({ ...form, username: e.target.value })} />
              </div>
              <div className="space-y-1">
                <Label>角色 <span className="text-destructive">*</span></Label>
                <Select value={form.role} onValueChange={(v) => { if (v) setForm({ ...form, role: v }) }}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="user">user</SelectItem>
                    <SelectItem value="admin">admin</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="space-y-1">
              <Label>密码 <span className="text-destructive">*</span></Label>
              <Input type="password" value={form.password} onChange={(e) => setForm({ ...form, password: e.target.value })} />
            </div>
            <div className="space-y-1">
              <Label>确认密码 <span className="text-destructive">*</span></Label>
              <Input type="password" value={form.confirmPwd} onChange={(e) => setForm({ ...form, confirmPwd: e.target.value })} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>取消</Button>
            <Button onClick={handleCreate} disabled={submitting}>
              {submitting ? "创建中..." : "创建"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 重置密码弹窗 */}
      <Dialog open={resetOpen} onOpenChange={setResetOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>重置密码 — {resetTarget?.username}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <Label>新密码 <span className="text-destructive">*</span></Label>
              <Input type="password" value={resetForm.password} onChange={(e) => setResetForm({ ...resetForm, password: e.target.value })} />
            </div>
            <div className="space-y-1">
              <Label>确认新密码 <span className="text-destructive">*</span></Label>
              <Input type="password" value={resetForm.confirmPwd} onChange={(e) => setResetForm({ ...resetForm, confirmPwd: e.target.value })} />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setResetOpen(false)}>取消</Button>
            <Button onClick={handleReset} disabled={submitting}>
              {submitting ? "重置中..." : "重置密码"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
