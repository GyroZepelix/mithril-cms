import { useLocation } from "react-router";
import { LogOut } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";

function buildBreadcrumb(pathname: string): string[] {
  const segments = pathname.replace(/^\/admin\/?/, "").split("/").filter(Boolean);
  return segments.map((s) => s.charAt(0).toUpperCase() + s.slice(1).replace(/-/g, " "));
}

export function Header() {
  const { state, logout } = useAuth();
  const location = useLocation();
  const breadcrumb = buildBreadcrumb(location.pathname);

  const adminEmail =
    state.status === "authenticated" ? state.admin.email : "";

  return (
    <header className="flex h-14 items-center justify-between border-b px-6">
      <nav className="flex items-center gap-1 text-sm text-muted-foreground">
        {breadcrumb.length === 0 ? (
          <span>Dashboard</span>
        ) : (
          breadcrumb.map((segment, i) => (
            <span key={i} className="flex items-center gap-1">
              {i > 0 && <span>/</span>}
              <span
                className={
                  i === breadcrumb.length - 1
                    ? "font-medium text-foreground"
                    : ""
                }
              >
                {segment}
              </span>
            </span>
          ))
        )}
      </nav>

      <div className="flex items-center gap-4">
        <span className="text-sm text-muted-foreground">{adminEmail}</span>
        <Button variant="ghost" size="sm" onClick={logout}>
          <LogOut className="h-4 w-4" />
          Logout
        </Button>
      </div>
    </header>
  );
}
