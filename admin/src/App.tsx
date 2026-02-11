import { BrowserRouter, Routes, Route, Navigate } from "react-router";
import { AuthProvider } from "@/lib/auth";
import { AppLayout } from "@/components/layout/AppLayout";
import { LoginPage } from "@/pages/LoginPage";
import { ContentListPage } from "@/pages/ContentListPage";
import { ContentEditPage } from "@/pages/ContentEditPage";
import { MediaPage } from "@/pages/MediaPage";
import { AuditLogPage } from "@/pages/AuditLogPage";
import { NotFoundPage } from "@/pages/NotFoundPage";

export function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/admin/login" element={<LoginPage />} />

          <Route path="/admin" element={<AppLayout />}>
            <Route index element={<Navigate to="/admin/content" replace />} />
            <Route path="content/:type" element={<ContentListPage />} />
            <Route path="content/:type/new" element={<ContentEditPage />} />
            <Route path="content/:type/:id" element={<ContentEditPage />} />
            <Route path="media" element={<MediaPage />} />
            <Route path="audit-log" element={<AuditLogPage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Route>

          <Route path="*" element={<Navigate to="/admin" replace />} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}
