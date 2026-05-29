import { useAuth } from "@/hooks/useAuth"

export function AdminRoute({ children }: { children: React.ReactNode }) {
  const { role } = useAuth()
  if (role !== "admin") {
    return (
      <div className="flex items-center justify-center h-screen">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-destructive mb-2">403</h1>
          <p className="text-muted-foreground">需要管理员权限</p>
        </div>
      </div>
    )
  }
  return <>{children}</>
}
