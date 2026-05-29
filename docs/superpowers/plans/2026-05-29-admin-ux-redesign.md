# 管理页面交互优化 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复管理页面交互断点——审批人下拉选择、参数白名单动态表单、Toast 通知、API Key 管理操作、用户密码确认。

**Architecture:** 后端新增 3 个管理端点（API Key 启用/禁用/删除、用户重置密码），Login 响应增加 username 字段。前端新增 Toast/Switch 两个共享组件，重写三个管理页面。

**Tech Stack:** Go/Gin（后端）、React + TypeScript + Tailwind CSS（前端）、shadcn/ui 组件库

---

## 文件变更清单

### 后端
| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/store/store.go` | 修改 | 新增 UpdateAPIKey、DeleteAPIKey、UpdateUserPassword |
| `internal/api/admin_handler.go` | 修改 | 新增 UpdateAPIKey、DeleteAPIKey、ResetPassword handler |
| `internal/api/human_handler.go` | 修改 | Login 响应增加 username |
| `internal/api/router.go` | 修改 | 注册 3 个新路由 |

### 前端
| 文件 | 操作 | 职责 |
|------|------|------|
| `web/src/components/ui/switch.tsx` | 新建 | Switch 开关组件 |
| `web/src/components/ui/sonner.tsx` | 新建 | Toast 通知组件（基于 sonner） |
| `web/src/lib/auth.ts` | 修改 | UserInfo 增加 username 字段 |
| `web/src/hooks/useAuth.tsx` | 修改 | 暴露 username，Login 存储 username |
| `web/src/components/Layout.tsx` | 修改 | Header 显示头像+用户名+角色标签 |
| `web/src/pages/ProgramPage.tsx` | 重写 | 审批人下拉、参数动态表单、表格信息可读化 |
| `web/src/pages/UserPage.tsx` | 重写 | 密码确认、头像列、重置密码、当前用户高亮 |
| `web/src/pages/ApiKeyPage.tsx` | 重写 | Key 脱敏显示、启用/禁用/删除操作 |
| `web/src/App.tsx` | 修改 | 添加 Toaster 组件 |
| `web/package.json` | 修改 | 添加 sonner 依赖 |

---

## Task 1: 后端 — Login 响应增加 username

**Files:**
- Modify: `internal/api/human_handler.go:45`

- [ ] **Step 1: 修改 Login handler 响应，增加 username**

在 `internal/api/human_handler.go` 的 `Login` 方法中，将 `c.JSON(http.StatusOK, gin.H{"token": tok, "role": u.Role, "user_id": u.ID})` 改为：

```go
c.JSON(http.StatusOK, gin.H{"token": tok, "role": u.Role, "user_id": u.ID, "username": u.Username})
```

- [ ] **Step 2: 提交**

```bash
git add internal/api/human_handler.go
git commit -m "feat: login 响应增加 username 字段"
```

---

## Task 2: 后端 — Store 层新增方法

**Files:**
- Modify: `internal/store/store.go:48` (APIKey 方法区域之后)

- [ ] **Step 1: 新增 UpdateAPIKey、DeleteAPIKey、UpdateUserPassword 方法**

在 `internal/store/store.go` 的 `GetAPIKeyByHash` 方法之后添加：

```go
func (s *Store) UpdateAPIKey(k *model.APIKey) error { return s.db.Save(k).Error }
func (s *Store) DeleteAPIKey(id uint64) error { return s.db.Delete(&model.APIKey{}, id).Error }
```

在 `GetUser` 方法之后添加：

```go
func (s *Store) UpdateUserPassword(id uint64, hash string) error {
	return s.db.Model(&model.User{}).Where("id = ?", id).Update("password_hash", hash).Error
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/store/store.go
git commit -m "feat: store 层新增 UpdateAPIKey/DeleteAPIKey/UpdateUserPassword"
```

---

## Task 3: 后端 — 新增 Admin handler 方法

**Files:**
- Modify: `internal/api/admin_handler.go:143` (文件末尾)

- [ ] **Step 1: 新增 UpdateAPIKey handler**

在 `internal/api/admin_handler.go` 的 `UpdateProgram` 方法之后添加：

```go
func (h *AdminHandler) UpdateAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var k model.APIKey
	if err := h.store.DB().First(&k, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API Key 不存在"})
		return
	}
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Enabled != nil {
		k.Enabled = *req.Enabled
	}
	if err := h.store.UpdateAPIKey(&k); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, k)
}

func (h *AdminHandler) DeleteAPIKey(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.store.DeleteAPIKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (h *AdminHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	hash, _ := auth.HashPassword(req.Password)
	if err := h.store.UpdateUserPassword(id, hash); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码已重置"})
}
```

注意：需要在文件顶部 import 中确认已有 `"LoopGuard/internal/auth"` 和 `"strconv"` 和 `"net/http"`（这些已存在）。

- [ ] **Step 2: 提交**

```bash
git add internal/api/admin_handler.go
git commit -m "feat: admin handler 新增 UpdateAPIKey/DeleteAPIKey/ResetPassword"
```

---

## Task 4: 后端 — 注册新路由

**Files:**
- Modify: `internal/api/router.go:61`

- [ ] **Step 1: 在 adminGrp 中添加 3 个新路由**

在 `internal/api/router.go` 的 `adminGrp.GET("/api-keys", admin.ListAPIKeys)` 行之后添加：

```go
		adminGrp.PUT("/api-keys/:id", admin.UpdateAPIKey)
		adminGrp.DELETE("/api-keys/:id", admin.DeleteAPIKey)
		adminGrp.PUT("/users/:id/password", admin.ResetPassword)
```

- [ ] **Step 2: 验证编译**

```bash
cd /Users/xujiangze/go/src/LoopGuard && go build ./...
```

Expected: 编译成功，无错误。

- [ ] **Step 3: 提交**

```bash
git add internal/api/router.go
git commit -m "feat: 注册 API Key 管理和用户重置密码路由"
```

---

## Task 5: 前端 — 安装 sonner 并创建 Toast 组件

**Files:**
- Modify: `web/package.json` (通过 npm install)
- Create: `web/src/components/ui/sonner.tsx`
- Modify: `web/src/App.tsx`

- [ ] **Step 1: 安装 sonner**

```bash
cd /Users/xujiangze/go/src/LoopGuard/web && npm install sonner
```

- [ ] **Step 2: 创建 Sonner 组件包装**

创建 `web/src/components/ui/sonner.tsx`：

```tsx
import { Toaster as Sonner } from "sonner"

type ToasterProps = React.ComponentProps<typeof Sonner>

function Toaster({ ...props }: ToasterProps) {
  return (
    <Sonner
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
      }}
      {...props}
    />
  )
}

export { Toaster }
```

- [ ] **Step 3: 在 App.tsx 中添加 Toaster**

在 `web/src/App.tsx` 中，导入并在 HashRouter 内部的最外层添加 `<Toaster />`。读取当前 App.tsx 内容，在 `<HashRouter>` 内部的第一个子元素外层（或同级）添加：

```tsx
import { Toaster } from "@/components/ui/sonner"
```

在 HashRouter 内部的路由结构之外（与 Routes 同级）添加：

```tsx
<Toaster richColors position="top-right" />
```

具体位置：用 `<Fragment>` 或 `<div>` 包裹 HashRouter 内部内容，使 Toaster 和 Routes 并列。

- [ ] **Step 4: 提交**

```bash
git add web/package.json web/package-lock.json web/src/components/ui/sonner.tsx web/src/App.tsx
git commit -m "feat: 添加 Toast 通知组件(sonner)"
```

---

## Task 6: 前端 — 创建 Switch 组件

**Files:**
- Create: `web/src/components/ui/switch.tsx`

- [ ] **Step 1: 创建 Switch 组件**

创建 `web/src/components/ui/switch.tsx`：

```tsx
import * as React from "react"
import * as SwitchPrimitives from "@radix-ui/react-switch"
import { cn } from "@/lib/utils"

const Switch = React.forwardRef<
  React.ComponentRef<typeof SwitchPrimitives.Root>,
  React.ComponentPropsWithoutRef<typeof SwitchPrimitives.Root>
>(({ className, ...props }, ref) => (
  <SwitchPrimitives.Root
    className={cn(
      "peer inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:bg-primary data-[state=unchecked]:bg-input",
      className,
    )}
    {...props}
    ref={ref}
  >
    <SwitchPrimitives.Thumb
      className={cn(
        "pointer-events-none block h-4 w-4 rounded-full bg-background shadow-lg ring-0 transition-transform data-[state=checked]:translate-x-4 data-[state=unchecked]:translate-x-0",
      )}
    />
  </SwitchPrimitives.Root>
))
Switch.displayName = SwitchPrimitives.Root.displayName

export { Switch }
```

- [ ] **Step 2: 安装 radix-ui switch 依赖**

```bash
cd /Users/xujiangze/go/src/LoopGuard/web && npm install @radix-ui/react-switch
```

- [ ] **Step 3: 提交**

```bash
git add web/src/components/ui/switch.tsx web/package.json web/package-lock.json
git commit -m "feat: 添加 Switch 开关组件"
```

---

## Task 7: 前端 — Auth 层增加 username

**Files:**
- Modify: `web/src/lib/auth.ts`
- Modify: `web/src/hooks/useAuth.tsx`

- [ ] **Step 1: 修改 auth.ts 的 UserInfo 接口**

在 `web/src/lib/auth.ts` 中，将 `UserInfo` 接口改为：

```ts
interface UserInfo {
  user_id: number
  role: string
  username: string
}
```

- [ ] **Step 2: 修改 useAuth.tsx 暴露 username**

在 `web/src/hooks/useAuth.tsx` 中：

1. 修改 `AuthState` 接口，增加 `username`：

```tsx
interface AuthState {
  userId: number | null
  role: Role | null
  username: string | null
  loggedIn: boolean
}
```

2. 修改 `getInitialState` 函数：

```tsx
function getInitialState(): AuthState {
  const user = getUser()
  const token = getToken()
  if (user && token) {
    return { userId: user.user_id, role: user.role as Role, username: user.username || null, loggedIn: true }
  }
  return { userId: null, role: null, username: null, loggedIn: false }
}
```

3. 修改 `login` 回调中的 `saveUser` 和 `setState`：

```tsx
const login = useCallback(async (username: string, password: string) => {
  const data = await api.post<{ token: string; role: Role; user_id: number; username: string }>("/auth/login", {
    username,
    password,
  })
  setToken(data.token)
  saveUser({ user_id: data.user_id, role: data.role, username: data.username })
  setState({ userId: data.user_id, role: data.role, username: data.username, loggedIn: true })
}, [])
```

4. 修改 `logout` 回调：

```tsx
const logout = useCallback(() => {
  clearToken()
  setState({ userId: null, role: null, username: null, loggedIn: false })
}, [])
```

5. 修改 `AuthContextValue` 接口：

```tsx
interface AuthContextValue extends AuthState {
  login: (username: string, password: string) => Promise<void>
  logout: () => void
}
```

（`AuthContextValue` 已经 extends `AuthState`，新增的 `username` 字段会自动包含，无需额外修改此接口）

- [ ] **Step 3: 提交**

```bash
git add web/src/lib/auth.ts web/src/hooks/useAuth.tsx
git commit -m "feat: auth 层增加 username 支持"
```

---

## Task 8: 前端 — Layout Header 改版

**Files:**
- Modify: `web/src/components/Layout.tsx`

- [ ] **Step 1: 修改 Header 显示头像+用户名+角色标签**

在 `web/src/components/Layout.tsx` 中：

1. 从 `useAuth` 中额外解构 `username`：

```tsx
const { role, username, logout } = useAuth()
```

2. 添加获取头像首字的辅助函数（在组件内部）：

```tsx
const initial = username ? username.charAt(0).toUpperCase() : "?"
```

3. 替换 header 内容（`<header>` 标签内的 `span` 和 `Button`）为：

```tsx
<header className="h-12 border-b flex items-center justify-end px-4 gap-3">
  <div className="flex items-center gap-2">
    <div className="w-7 h-7 rounded-full bg-primary/10 text-primary flex items-center justify-center text-xs font-semibold">
      {initial}
    </div>
    <span className="text-sm font-medium">{username}</span>
    <Badge variant="secondary" className="text-[10px] px-1.5 py-0">{role}</Badge>
  </div>
  <Button variant="ghost" size="sm" onClick={logout}>
    退出
  </Button>
</header>
```

4. 确保 Badge 已导入：

```tsx
import { Badge } from "@/components/ui/badge"
```

- [ ] **Step 2: 提交**

```bash
git add web/src/components/Layout.tsx
git commit -m "feat: Layout Header 显示头像+用户名+角色标签"
```

---

## Task 9: 前端 — 重写 ProgramPage

**Files:**
- Rewrite: `web/src/pages/ProgramPage.tsx`

这是最大的改动。完整重写，包含：
- 表格：审批人显示名字、二进制路径副行、超时列、启用/禁用操作
- 创建弹窗：分三区表单、审批人下拉、参数白名单动态 key-value
- 编辑弹窗：Switch 组件、审批人下拉
- 按钮 loading 态
- Toast 通知

- [ ] **Step 1: 重写 ProgramPage.tsx**

完整替换 `web/src/pages/ProgramPage.tsx` 为以下内容：

```tsx
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
```

- [ ] **Step 2: 验证编译**

```bash
cd /Users/xujiangze/go/src/LoopGuard/web && npx tsc --noEmit
```

Expected: 无类型错误。

- [ ] **Step 3: 提交**

```bash
git add web/src/pages/ProgramPage.tsx
git commit -m "feat: 程序管理页交互优化（审批人下拉、参数动态表单、表格改版）"
```

---

## Task 10: 前端 — 重写 UserPage

**Files:**
- Rewrite: `web/src/pages/UserPage.tsx`

- [ ] **Step 1: 重写 UserPage.tsx**

完整替换 `web/src/pages/UserPage.tsx` 为以下内容：

```tsx
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
```

- [ ] **Step 2: 提交**

```bash
git add web/src/pages/UserPage.tsx
git commit -m "feat: 用户管理页交互优化（密码确认、重置密码、当前用户高亮）"
```

---

## Task 11: 前端 — 重写 ApiKeyPage

**Files:**
- Rewrite: `web/src/pages/ApiKeyPage.tsx`

- [ ] **Step 1: 重写 ApiKeyPage.tsx**

完整替换 `web/src/pages/ApiKeyPage.tsx` 为以下内容：

```tsx
import { useEffect, useState } from "react"
import { api } from "@/lib/api"
import type { APIKey } from "@/types"
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
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogCancel,
  AlertDialogAction,
} from "@/components/ui/alert-dialog"
import { toast } from "sonner"

function maskKey(name: string): string {
  if (name.length <= 10) return name
  return name.slice(0, 6) + "..." + name.slice(-4)
}

export function ApiKeyPage() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [name, setName] = useState("")

  const [plainKey, setPlainKey] = useState("")
  const [showKey, setShowKey] = useState(false)
  const [copied, setCopied] = useState(false)

  const [deleteTarget, setDeleteTarget] = useState<APIKey | null>(null)
  const [deleteOpen, setDeleteOpen] = useState(false)

  const fetchKeys = () => {
    setLoading(true)
    api
      .get<APIKey[]>("/api-keys")
      .then(setKeys)
      .catch(() => setKeys([]))
      .finally(() => setLoading(false))
  }

  useEffect(() => { fetchKeys() }, [])

  const handleCreate = async () => {
    if (!name.trim()) {
      toast.error("请输入 Key 名称")
      return
    }
    setSubmitting(true)
    try {
      const data = await api.post<{ id: number; name: string; api_key: string }>("/api-keys", { name: name.trim() })
      toast.success("API Key 创建成功")
      setCreateOpen(false)
      setName("")
      setPlainKey(data.api_key)
      setShowKey(true)
      setCopied(false)
      fetchKeys()
    } catch (err) {
      toast.error("创建失败", { description: err instanceof Error ? err.message : "未知错误" })
    } finally {
      setSubmitting(false)
    }
  }

  const handleCopy = async () => {
    await navigator.clipboard.writeText(plainKey)
    setCopied(true)
    toast.success("已复制到剪贴板")
  }

  const toggleEnabled = async (k: APIKey) => {
    try {
      await api.put(`/api-keys/${k.id}`, { enabled: !k.enabled })
      toast.success(k.enabled ? "已禁用" : "已启用", { description: k.name })
      fetchKeys()
    } catch (err) {
      toast.error("操作失败", { description: err instanceof Error ? err.message : "未知错误" })
    }
  }

  const confirmDelete = async () => {
    if (!deleteTarget) return
    try {
      await api.del(`/api-keys/${deleteTarget.id}`)
      toast.success("已删除", { description: deleteTarget.name })
      setDeleteOpen(false)
      setDeleteTarget(null)
      fetchKeys()
    } catch (err) {
      toast.error("删除失败", { description: err instanceof Error ? err.message : "未知错误" })
    }
  }

  if (loading) return <p className="text-muted-foreground text-sm">加载中...</p>

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold">API Key 管理</h1>
          <p className="text-sm text-muted-foreground mt-1">用于 AI agent 连接 LoopGuard 的服务账号密钥</p>
        </div>
        <Button onClick={() => setCreateOpen(true)}>创建 Key</Button>
      </div>

      {keys.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center">暂无 API Key</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>名称</TableHead>
              <TableHead>Key</TableHead>
              <TableHead className="w-20">状态</TableHead>
              <TableHead className="w-44">创建时间</TableHead>
              <TableHead className="w-28">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {keys.map((k) => (
              <TableRow key={k.id} className={!k.enabled ? "bg-red-50 dark:bg-red-950/20" : undefined}>
                <TableCell className="font-medium">{k.name}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  {k.id ? maskKey(`lg_${k.id}`) : "—"}
                </TableCell>
                <TableCell>
                  <Badge variant={k.enabled ? "default" : "destructive"}>
                    {k.enabled ? "启用" : "已禁用"}
                  </Badge>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {new Date(k.created_at).toLocaleString("zh-CN")}
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      className={k.enabled ? "text-destructive" : "text-green-600"}
                      onClick={() => toggleEnabled(k)}
                    >
                      {k.enabled ? "禁用" : "启用"}
                    </Button>
                    {!k.enabled && (
                      <Button variant="ghost" size="sm" className="text-destructive" onClick={() => { setDeleteTarget(k); setDeleteOpen(true) }}>
                        删除
                      </Button>
                    )}
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {/* 创建弹窗 */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>创建 API Key</DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <Label>名称 <span className="text-destructive">*</span></Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如 hermes-agent" />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>取消</Button>
            <Button onClick={handleCreate} disabled={submitting}>
              {submitting ? "创建中..." : "创建"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 明文 Key 展示弹窗 */}
      <AlertDialog open={showKey} onOpenChange={setShowKey}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>请立即复制，此密钥只显示一次</AlertDialogTitle>
            <AlertDialogDescription>关闭此弹窗后将无法再次查看此密钥的明文。</AlertDialogDescription>
            <div className="space-y-3 mt-3">
              <div className="bg-muted rounded-md p-3 font-mono text-sm break-all select-all">{plainKey}</div>
              <Button variant="outline" className="w-full" onClick={handleCopy}>
                {copied ? "已复制" : "复制到剪贴板"}
              </Button>
            </div>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setShowKey(false)}>我已保存，关闭</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* 删除确认弹窗 */}
      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认删除 API Key？</AlertDialogTitle>
            <AlertDialogDescription asChild>
              <div>
                <p>Key: <strong>{deleteTarget?.name}</strong></p>
                <p className="mt-2 text-destructive font-medium">此操作不可撤销。使用该 Key 的 AI agent 将立即失去访问权限。</p>
              </div>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeleteOpen(false)}>取消</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive text-destructive-foreground hover:bg-destructive/90" onClick={confirmDelete}>
              确认删除
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
```

- [ ] **Step 2: 检查 alert-dialog 组件是否导出 AlertDialogCancel**

读取 `web/src/components/ui/alert-dialog.tsx`，确认 `AlertDialogCancel` 已导出。如果没有，需要添加导出。

- [ ] **Step 3: 提交**

```bash
git add web/src/pages/ApiKeyPage.tsx
git commit -m "feat: API Key 管理页交互优化（启用/禁用/删除、脱敏显示、删除确认）"
```

---

## Task 12: 验证

- [ ] **Step 1: 后端编译检查**

```bash
cd /Users/xujiangze/go/src/LoopGuard && go build ./...
```

Expected: 编译成功。

- [ ] **Step 2: 前端类型检查**

```bash
cd /Users/xujiangze/go/src/LoopGuard/web && npx tsc --noEmit
```

Expected: 无类型错误。

- [ ] **Step 3: 前端构建检查**

```bash
cd /Users/xujiangze/go/src/LoopGuard/web && npm run build
```

Expected: 构建成功，无错误。

---

## Self-Review

### Spec Coverage

| 设计文档要求 | 对应 Task |
|-------------|-----------|
| 审批人下拉选择 | Task 9 (ProgramPage) |
| 参数白名单动态 key-value 表单 | Task 9 (ProgramPage) |
| 表格审批人显示名字 | Task 9 (ProgramPage) |
| 表格启用/禁用快捷操作 | Task 9 (ProgramPage) |
| 编辑弹窗 Switch 组件 | Task 6 + Task 9 |
| Toast 通知 | Task 5 (全局 sonner) |
| 按钮 loading 态 | Task 9, 10, 11 |
| Header 头像+用户名+角色 | Task 7 + Task 8 |
| 用户创建密码确认 | Task 10 (UserPage) |
| 用户重置密码 | Task 3 + Task 10 |
| 当前用户高亮 | Task 10 (UserPage) |
| API Key 启用/禁用 | Task 3 + Task 4 + Task 11 |
| API Key 删除确认 | Task 3 + Task 4 + Task 11 |
| API Key 脱敏显示 | Task 11 (ApiKeyPage) |
| 空状态去掉开发者信息 | Task 10 (UserPage) |
| Login 响应增加 username | Task 1 |

### Placeholder Scan
无 TBD/TODO/待定内容。

### Type Consistency
- `UserInfo.username: string` 在 auth.ts 和 useAuth.tsx 中一致
- API 响应类型 `api.post<{..., username: string}>` 与 auth.ts 的 `UserInfo` 一致
- `ParamEntry` 接口仅在 ProgramPage 内部使用，不跨文件
