import { useEffect, useState } from "react";
import { NavLink } from "react-router";
import {
  FileText,
  Image,
  ScrollText,
  Loader2,
} from "lucide-react";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";

type ContentType = {
  name: string;
  display_name: string;
  entry_count: number;
};

export function Sidebar() {
  const [contentTypes, setContentTypes] = useState<ContentType[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function fetchContentTypes() {
      try {
        const types = await api.get<ContentType[]>("/admin/api/content-types");
        if (!cancelled) {
          setContentTypes(types);
        }
      } catch {
        // Silently fail -- sidebar will show empty
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    fetchContentTypes();
    return () => {
      cancelled = true;
    };
  }, []);

  const navLinkClass = ({ isActive }: { isActive: boolean }) =>
    cn(
      "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
      isActive
        ? "bg-sidebar-accent text-sidebar-accent-foreground"
        : "text-sidebar-foreground/70 hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
    );

  return (
    <aside className="flex h-full w-64 flex-col border-r border-sidebar-border bg-sidebar">
      <div className="flex h-14 items-center border-b border-sidebar-border px-4">
        <h1 className="text-lg font-bold text-sidebar-foreground">Mithril CMS</h1>
      </div>

      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        <div className="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-sidebar-foreground/50">
          Content
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-4">
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          </div>
        ) : contentTypes.length === 0 ? (
          <p className="px-3 text-sm text-muted-foreground">No content types</p>
        ) : (
          contentTypes.map((ct) => (
            <NavLink
              key={ct.name}
              to={`/admin/content/${ct.name}`}
              className={navLinkClass}
            >
              <FileText className="h-4 w-4" />
              {ct.display_name}
            </NavLink>
          ))
        )}

        <div className="mb-2 mt-6 px-3 text-xs font-semibold uppercase tracking-wider text-sidebar-foreground/50">
          System
        </div>

        <NavLink to="/admin/media" className={navLinkClass}>
          <Image className="h-4 w-4" />
          Media Library
        </NavLink>

        <NavLink to="/admin/audit-log" className={navLinkClass}>
          <ScrollText className="h-4 w-4" />
          Audit Log
        </NavLink>
      </nav>
    </aside>
  );
}
