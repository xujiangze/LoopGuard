import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import type { User } from "@/types"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
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

export function UserPage() {
  const [users, setUsers] = useState<User[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [error, setError] = useState("")
  const [form, setForm] = useState({ username: "", password: "", role: "user" as string })

  const fetchUsers = () => {
    setLoading(true)
    api
      .get<User[]>("/users")
      .then(setUsers)
      .catch(() => {
        // 后端可能没有 GET /users 列表接口，暂时忽略
        setUsers([])
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchUsers()
  }, [])

  const handleCreate = async () => {
    setError("")
    try {
      await api.post("/users", {
        username: form.username,
        password: form.password,
        role: form.role,
      })
      setCreateOpen(false)
      setForm({ username: "", password: "", role: "user" })
      fetchUsers()
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建失败")
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">用户管理</h1>
        <Button onClick={() => setCreateOpen(true)}>创建用户</Button>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}

      {loading ? (
        <p className="text-muted-foreground text-sm">加载中...</p>
      ) : users.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center">
          暂无用户数据（后端可能未提供用户列表接口）
        </p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-20">ID</TableHead>
              <TableHead>用户名</TableHead>
              <TableHead className="w-24">角色</TableHead>
              <TableHead className="w-44">创建时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {users.map((u) => (
              <TableRow key={u.id}>
                <TableCell className="font-mono">{u.id}</TableCell>
                <TableCell>{u.username}</TableCell>
                <TableCell>{u.role}</TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {new Date(u.created_at).toLocaleString("zh-CN")}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>创建用户</DialogTitle>
          </DialogHeader>
          <div className="space-y-3">
            <div className="space-y-1">
              <Label>用户名</Label>
              <Input
                value={form.username}
                onChange={(e) => setForm({ ...form, username: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>密码</Label>
              <Input
                type="password"
                value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <Label>角色</Label>
              <Select value={form.role} onValueChange={(v) => { if (v) setForm({ ...form, role: v }) }}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="user">user</SelectItem>
                  <SelectItem value="admin">admin</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              取消
            </Button>
            <Button onClick={handleCreate}>创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
