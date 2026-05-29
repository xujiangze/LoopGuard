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
            <AlertDialogDescription>
              Key: <strong>{deleteTarget?.name}</strong>。此操作不可撤销，使用该 Key 的 AI agent 将立即失去访问权限。
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
