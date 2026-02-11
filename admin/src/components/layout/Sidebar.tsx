import { useEffect, useRef, useState, useCallback } from "react";
import { NavLink } from "react-router";
import {
  FileText,
  Image,
  ScrollText,
  Loader2,
  RefreshCw,
  CheckCircle2,
  AlertTriangle,
} from "lucide-react";
import { api, ApiRequestError } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";
import type { SchemaRefreshResult, ValidationDetail } from "@/lib/types";

type ContentType = {
  name: string;
  display_name: string;
  entry_count: number;
};

type RefreshStatus = "idle" | "loading" | "success" | "error";

export function Sidebar() {
  const [contentTypes, setContentTypes] = useState<ContentType[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshKey, setRefreshKey] = useState(0);

  // Schema refresh state
  const [refreshStatus, setRefreshStatus] = useState<RefreshStatus>("idle");
  const [refreshMessage, setRefreshMessage] = useState("");
  const [breakingChanges, setBreakingChanges] = useState<ValidationDetail[]>([]);
  const [dialogOpen, setDialogOpen] = useState(false);

  // Timer refs to prevent leaks
  const successTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const errorTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);

  useEffect(() => {
    return () => {
      clearTimeout(successTimerRef.current);
      clearTimeout(errorTimerRef.current);
    };
  }, []);

  const fetchContentTypes = useCallback(async (signal?: AbortSignal) => {
    try {
      const types = await api.get<ContentType[]>("/admin/api/content-types");
      if (!signal?.aborted) {
        setContentTypes(types);
      }
    } catch {
      // Silently fail -- sidebar will show empty
    } finally {
      if (!signal?.aborted) {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    fetchContentTypes(controller.signal);
    return () => controller.abort();
  }, [fetchContentTypes, refreshKey]);

  async function handleSchemaRefresh() {
    clearTimeout(successTimerRef.current);
    clearTimeout(errorTimerRef.current);
    setRefreshStatus("loading");
    setRefreshMessage("");
    setBreakingChanges([]);

    try {
      const result = await api.post<SchemaRefreshResult>("/admin/api/schema/refresh");
      setRefreshStatus("success");
      setRefreshMessage(result.message);
      // Refresh the content types list
      setRefreshKey((k) => k + 1);
      // Auto-clear success message after 3 seconds
      successTimerRef.current = setTimeout(() => {
        setRefreshStatus((prev) => (prev === "success" ? "idle" : prev));
        setRefreshMessage("");
      }, 3000);
    } catch (err) {
      setRefreshStatus("error");
      if (err instanceof ApiRequestError && err.status === 409) {
        // Breaking changes -- show dialog with details
        const body = err.body as { error?: { details?: ValidationDetail[]; message?: string } } | undefined;
        const details = body?.error?.details ?? [];
        setBreakingChanges(details);
        setRefreshMessage(body?.error?.message ?? "Schema refresh blocked by breaking changes");
        setDialogOpen(true);
      } else {
        setRefreshMessage(err instanceof Error ? err.message : "Schema refresh failed");
        // Auto-clear non-dialog errors after 5 seconds
        errorTimerRef.current = setTimeout(() => {
          setRefreshStatus((prev) => (prev === "error" ? "idle" : prev));
          setRefreshMessage("");
        }, 5000);
      }
    }
  }

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
        <div className="mb-2 flex items-center justify-between px-3">
          <span className="text-xs font-semibold uppercase tracking-wider text-sidebar-foreground/50">
            Content
          </span>
          <Button
            variant="ghost"
            size="icon"
            className="h-6 w-6"
            onClick={handleSchemaRefresh}
            disabled={refreshStatus === "loading"}
            title="Refresh schema from YAML files"
          >
            {refreshStatus === "loading" ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" />
            ) : refreshStatus === "success" ? (
              <CheckCircle2 className="h-3.5 w-3.5 text-green-600" />
            ) : refreshStatus === "error" ? (
              <AlertTriangle className="h-3.5 w-3.5 text-destructive" />
            ) : (
              <RefreshCw className="h-3.5 w-3.5" />
            )}
          </Button>
        </div>

        {/* Inline status message */}
        {refreshMessage && refreshStatus !== "idle" && !dialogOpen && (
          <p
            className={cn(
              "px-3 text-xs",
              refreshStatus === "success" ? "text-green-600" : "text-destructive",
            )}
          >
            {refreshMessage}
          </p>
        )}

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

      {/* Breaking changes dialog */}
      <Dialog
        open={dialogOpen}
        onOpenChange={(open) => {
          setDialogOpen(open);
          if (!open) {
            setRefreshMessage("");
            setBreakingChanges([]);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Schema Refresh Blocked</DialogTitle>
            <DialogDescription>
              {refreshMessage}
            </DialogDescription>
          </DialogHeader>
          {breakingChanges.length > 0 && (
            <div className="max-h-64 overflow-y-auto">
              <ul className="space-y-2">
                {breakingChanges.map((change, i) => (
                  <li
                    key={i}
                    className="rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm"
                  >
                    <span className="font-medium">{change.field}:</span>{" "}
                    {change.message}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </aside>
  );
}
