import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { api } from "@/lib/api"
import type { Ticket, TicketStatus } from "@/types"
import { TICKET_STATUS_LABELS } from "@/types"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"

const STATUS_FILTERS: { label: string; value: TicketStatus | "" }[] = [
  { label: "全部", value: "" },
  { label: "待审批", value: "pending_approval" },
  { label: "已完成", value: "done" },
  { label: "已驳回", value: "rejected" },
  { label: "执行失败", value: "exec_failed" },
  { label: "Dry-run 失败", value: "dryrun_failed" },
]

const STATUS_COLORS: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending_dryrun: "secondary",
  dryrun_failed: "outline",
  pending_approval: "default",
  approved: "secondary",
  executing: "secondary",
  done: "default",
  exec_failed: "destructive",
  rejected: "destructive",
}

export function TicketListPage() {
  const [status, setStatus] = useState<TicketStatus | "">("")
  const [tickets, setTickets] = useState<Ticket[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setLoading(true)
    const params = status ? `?status=${status}` : ""
    api
      .get<Ticket[]>(`/tickets${params}`)
      .then(setTickets)
      .catch(() => setTickets([]))
      .finally(() => setLoading(false))
  }, [status])

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-semibold">工单列表</h1>

      <Tabs value={status} onValueChange={(v) => setStatus(v as TicketStatus | "")}>
        <TabsList>
          {STATUS_FILTERS.map((f) => (
            <TabsTrigger key={f.value} value={f.value}>
              {f.label}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      {loading ? (
        <p className="text-muted-foreground text-sm">加载中...</p>
      ) : tickets.length === 0 ? (
        <p className="text-muted-foreground text-sm py-8 text-center">暂无工单</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-20">ID</TableHead>
              <TableHead>程序 ID</TableHead>
              <TableHead className="w-28">状态</TableHead>
              <TableHead className="w-44">提交时间</TableHead>
              <TableHead className="w-20">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tickets.map((t) => (
              <TableRow key={t.id}>
                <TableCell className="font-mono">{t.id}</TableCell>
                <TableCell>{t.program_id}</TableCell>
                <TableCell>
                  <Badge variant={STATUS_COLORS[t.status] || "default"}>
                    {TICKET_STATUS_LABELS[t.status]}
                  </Badge>
                </TableCell>
                <TableCell className="text-muted-foreground text-sm">
                  {new Date(t.created_at).toLocaleString("zh-CN")}
                </TableCell>
                <TableCell>
                  <Link to={`/tickets/${t.id}`}>
                    <Button variant="ghost" size="sm">
                      查看
                    </Button>
                  </Link>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
