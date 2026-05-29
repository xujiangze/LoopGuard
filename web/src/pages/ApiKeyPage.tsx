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
  AlertDialogAction,
} from "@/components/ui/alert-dialog"

export function ApiKeyPage() {
  const [keys, setKeys] = useState<APIKey[]>([])
  const [loading, setLoading] = useState(true)
  const [createOpen, setCreateOpen] = useState(false)
  const [error, setError] = useState("")
  const [name, setName] = useState("")

  // 明文 key 展示
  const [plainKey, setPlainKey] = useState("")
  const [showKey, setShowKey] = useState(false)
  const [copied, setCopied] = useState(false)

  const fetchKeys = () => {
    setLoading(true)
    api
      .get<APIKey[]>("/api-keys")
      .then(setKeys)
      .catch(() => {
        setKeys([])
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchKeys()
  }, [])

  const handleCreate = async () => {
    setError("")
    try {
      const data = await api.post<{ id: number; name: string; api_key: string }>("/api-keys", {
        name,
      })
      setCreateOpen(false)
      setName("")
      setPlainKey(data.api_key)
      setShowKey(true)
      setCopied(false)
      fetchKeys()
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建失败")
    }
  }

  const handleCopy = async () => {
    await navigator.clipboard.writeText(plainKey)
    setCopied(true)
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold">API Key 管理</h1>
        <Button onClick={() => setCreateOpen(true)}>创建 Key</Button>
      </div>

      {error && <p className="text-sm text-destructive">{error}</p>}

      {loading ? (
        <p className="text-muted-foreground text-sm">加载中...</p>
      ) : keys.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center">暂无 API Key</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-20">ID</TableHead>
              <TableHead>名称</TableHead>
              <TableHead className="w-24">状态</TableHead>
              <TableHead className="w-44">创建时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {keys.map((k) => (
              <TableRow key={k.id}>
                <TableCell className="font-mono">{k.id}</TableCell>
                <TableCell>{k.name}</TableCell>
                <TableCell>
                  <Badge variant={k.enabled ? "default" : "outline"}>
                    {k.enabled ? "启用" : "禁用"}
                  </Badge>
                </TableCell>
                <TableCell className="text-sm text-muted-foreground">
                  {new Date(k.created_at).toLocaleString("zh-CN")}
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
            <Label>名称</Label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="如 hermes-agent" />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>
              取消
            </Button>
            <Button onClick={handleCreate}>创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 明文 Key 展示弹窗 */}
      <AlertDialog open={showKey} onOpenChange={setShowKey}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>请立即复制，此密钥只显示一次</AlertDialogTitle>
            <AlertDialogDescription>
              关闭此弹窗后将无法再次查看此密钥的明文。
            </AlertDialogDescription>
            <div className="space-y-3 mt-3">
              <div className="bg-muted rounded-md p-3 font-mono text-sm break-all select-all">
                {plainKey}
              </div>
              <Button
                variant="outline"
                className="w-full"
                onClick={handleCopy}
              >
                {copied ? "已复制" : "复制到剪贴板"}
              </Button>
            </div>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setShowKey(false)}>
              我已保存，关闭
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
