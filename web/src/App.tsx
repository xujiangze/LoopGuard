import { HashRouter, Routes, Route } from "react-router-dom"
import { AuthProvider } from "@/hooks/useAuth"
import { Layout } from "@/components/Layout"
import { ProtectedRoute } from "@/components/ProtectedRoute"
import { AdminRoute } from "@/components/AdminRoute"
import { LoginPage } from "@/pages/LoginPage"
import { TicketListPage } from "@/pages/TicketListPage"
import { TicketDetailPage } from "@/pages/TicketDetailPage"
import { ProgramPage } from "@/pages/ProgramPage"
import { UserPage } from "@/pages/UserPage"
import { ApiKeyPage } from "@/pages/ApiKeyPage"
import { Toaster } from "@/components/ui/sonner"

export default function App() {
  return (
    <AuthProvider>
      <HashRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            element={
              <ProtectedRoute>
                <Layout />
              </ProtectedRoute>
            }
          >
            <Route path="/" element={<TicketListPage />} />
            <Route path="/tickets/:id" element={<TicketDetailPage />} />
            <Route
              path="/admin/programs"
              element={
                <AdminRoute>
                  <ProgramPage />
                </AdminRoute>
              }
            />
            <Route
              path="/admin/users"
              element={
                <AdminRoute>
                  <UserPage />
                </AdminRoute>
              }
            />
            <Route
              path="/admin/api-keys"
              element={
                <AdminRoute>
                  <ApiKeyPage />
                </AdminRoute>
              }
            />
          </Route>
        </Routes>
        <Toaster richColors position="top-right" />
      </HashRouter>
    </AuthProvider>
  )
}
