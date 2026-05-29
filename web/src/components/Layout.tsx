import { Link, Outlet, useLocation } from "react-router-dom"
import { useAuth } from "@/hooks/useAuth"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Separator } from "@/components/ui/separator"

const NAV_ITEMS = [
  { label: "工单列表", path: "/" },
]

const ADMIN_ITEMS = [
  { label: "程序管理", path: "/admin/programs" },
  { label: "用户管理", path: "/admin/users" },
  { label: "API Key 管理", path: "/admin/api-keys" },
]

export function Layout() {
  const { role, username, logout } = useAuth()
  const location = useLocation()
  const initial = username ? username.charAt(0).toUpperCase() : "?"

  const isActive = (path: string) => location.pathname === path

  return (
    <div className="min-h-screen flex">
      <aside className="w-56 border-r bg-card flex flex-col">
        <div className="p-4 font-semibold text-lg border-b">LoopGuard</div>
        <nav className="flex-1 p-2 space-y-1">
          {NAV_ITEMS.map((item) => (
            <Link key={item.path} to={item.path}>
              <div
                className={`px-3 py-2 rounded-md text-sm ${
                  isActive(item.path)
                    ? "bg-primary text-primary-foreground"
                    : "hover:bg-accent text-foreground"
                }`}
              >
                {item.label}
              </div>
            </Link>
          ))}
          {role === "admin" && (
            <>
              <Separator className="my-2" />
              <div className="px-3 py-1 text-xs text-muted-foreground font-medium">管理</div>
              {ADMIN_ITEMS.map((item) => (
                <Link key={item.path} to={item.path}>
                  <div
                    className={`px-3 py-2 rounded-md text-sm ${
                      isActive(item.path)
                        ? "bg-primary text-primary-foreground"
                        : "hover:bg-accent text-foreground"
                    }`}
                  >
                    {item.label}
                  </div>
                </Link>
              ))}
            </>
          )}
        </nav>
      </aside>
      <div className="flex-1 flex flex-col">
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
        <main className="flex-1 p-6 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
