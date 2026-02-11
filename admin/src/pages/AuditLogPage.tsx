import { useState, useEffect } from "react";
import { useSearchParams } from "react-router";
import { ChevronDown, ChevronRight, Loader2, Filter, X } from "lucide-react";
import { api } from "@/lib/api";
import { formatDateLong } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Pagination } from "@/components/Pagination";
import type { AuditLogEntry, PaginationMeta } from "@/lib/types";

const DEFAULT_PER_PAGE = 50;

const ACTION_OPTIONS = [
  { value: "", label: "All actions" },
  { value: "entry.create", label: "Entry Create" },
  { value: "entry.update", label: "Entry Update" },
  { value: "entry.publish", label: "Entry Publish" },
  { value: "schema.refresh", label: "Schema Refresh" },
  { value: "admin.login.success", label: "Login Success" },
  { value: "admin.login.failure", label: "Login Failure" },
  { value: "media.upload", label: "Media Upload" },
  { value: "media.delete", label: "Media Delete" },
] as const;

/** Map action strings to visual badge variants. */
function actionBadgeVariant(action: string): "default" | "secondary" | "destructive" | "outline" {
  if (action.includes("delete") || action.includes("failure")) return "destructive";
  if (action.includes("create") || action.includes("upload")) return "default";
  if (action.includes("publish") || action.includes("refresh")) return "outline";
  return "secondary";
}

export function AuditLogPage() {
  const [searchParams, setSearchParams] = useSearchParams();

  const page = Number(searchParams.get("page")) || 1;
  const perPage = Math.min(Math.max(Number(searchParams.get("per_page")) || DEFAULT_PER_PAGE, 1), 100);
  const actionFilter = searchParams.get("action") || "";
  const resourceFilter = searchParams.get("resource") || "";

  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  // Local filter input state for resource (applied on submit)
  const [resourceInput, setResourceInput] = useState(resourceFilter);

  // Sync resourceInput when URL changes (e.g. browser back/forward)
  useEffect(() => {
    setResourceInput(resourceFilter);
  }, [resourceFilter]);

  useEffect(() => {
    let cancelled = false;

    async function fetchLogs() {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({
          page: String(page),
          per_page: String(perPage),
        });
        if (actionFilter) params.set("action", actionFilter);
        if (resourceFilter) params.set("resource", resourceFilter);

        const result = await api.getWithMeta<AuditLogEntry[], PaginationMeta>(
          `/admin/api/audit-log?${params.toString()}`
        );
        if (!cancelled) {
          setEntries(result.data);
          setMeta(result.meta);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load audit log");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchLogs();
    return () => { cancelled = true; };
  }, [page, perPage, actionFilter, resourceFilter]);

  function setFilter(key: string, value: string) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (value) {
        next.set(key, value);
      } else {
        next.delete(key);
      }
      next.set("page", "1");
      return next;
    });
  }

  function handlePageChange(newPage: number) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.set("page", String(newPage));
      return next;
    });
  }

  function handleResourceSubmit(e: React.FormEvent) {
    e.preventDefault();
    setFilter("resource", resourceInput.trim());
  }

  function clearFilters() {
    setResourceInput("");
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      next.delete("action");
      next.delete("resource");
      next.set("page", "1");
      return next;
    });
  }

  const hasFilters = actionFilter || resourceFilter;

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold">Audit Log</h1>

      {/* Filters */}
      <div className="mb-4 flex flex-wrap items-end gap-3">
        <div className="flex flex-col gap-1">
          <label htmlFor="action-filter" className="text-xs font-medium text-muted-foreground">
            Action
          </label>
          <select
            id="action-filter"
            value={actionFilter}
            onChange={(e) => setFilter("action", e.target.value)}
            className="h-9 rounded-md border border-input bg-background px-3 text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
          >
            {ACTION_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>

        <form onSubmit={handleResourceSubmit} className="flex flex-col gap-1">
          <label htmlFor="resource-filter" className="text-xs font-medium text-muted-foreground">
            Resource
          </label>
          <div className="flex items-center gap-2">
            <Input
              id="resource-filter"
              placeholder="e.g. blog_posts"
              value={resourceInput}
              onChange={(e) => setResourceInput(e.target.value)}
              className="h-9 w-48"
            />
            <Button type="submit" variant="outline" size="sm">
              <Filter className="h-3.5 w-3.5" />
              Filter
            </Button>
          </div>
        </form>

        {hasFilters && (
          <Button variant="ghost" size="sm" onClick={clearFilters} className="mb-0.5">
            <X className="h-3.5 w-3.5" />
            Clear filters
          </Button>
        )}

        {meta && (
          <p className="mb-1 ml-auto text-sm text-muted-foreground">
            {meta.total} {meta.total === 1 ? "entry" : "entries"}
          </p>
        )}
      </div>

      {/* Error */}
      {error && (
        <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* Table */}
      {loading && entries.length === 0 ? (
        <AuditLogSkeleton />
      ) : entries.length === 0 && !loading ? (
        <div className="flex flex-col items-center justify-center rounded-md border border-dashed py-16">
          <p className="text-muted-foreground">
            {hasFilters ? "No audit log entries match your filters." : "No audit log entries yet."}
          </p>
        </div>
      ) : (
        <>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-10" />
                  <TableHead>Timestamp</TableHead>
                  <TableHead>Action</TableHead>
                  <TableHead>Actor</TableHead>
                  <TableHead>Resource</TableHead>
                  <TableHead>Resource ID</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center">
                      <Loader2 className="mx-auto h-4 w-4 animate-spin" />
                    </TableCell>
                  </TableRow>
                )}
                {!loading && entries.map((entry) => (
                  <AuditLogRow
                    key={entry.id}
                    entry={entry}
                    expanded={expandedId === entry.id}
                    onToggle={() => setExpandedId(expandedId === entry.id ? null : entry.id)}
                  />
                ))}
              </TableBody>
            </Table>
          </div>

          {meta && meta.total_pages > 1 && (
            <Pagination
              page={meta.page}
              totalPages={meta.total_pages}
              onPageChange={handlePageChange}
            />
          )}
        </>
      )}
    </div>
  );
}

function AuditLogRow({
  entry,
  expanded,
  onToggle,
}: {
  entry: AuditLogEntry;
  expanded: boolean;
  onToggle: () => void;
}) {
  const hasPayload = entry.payload !== null && entry.payload !== undefined;
  const ChevronIcon = expanded ? ChevronDown : ChevronRight;

  return (
    <>
      <TableRow
        className={hasPayload ? "cursor-pointer" : ""}
        onClick={hasPayload ? onToggle : undefined}
      >
        <TableCell className="w-10 px-2">
          {hasPayload && (
            <ChevronIcon className="h-4 w-4 text-muted-foreground" />
          )}
        </TableCell>
        <TableCell className="whitespace-nowrap text-muted-foreground">
          {formatDateLong(entry.created_at)}
        </TableCell>
        <TableCell>
          <Badge variant={actionBadgeVariant(entry.action)}>
            {entry.action}
          </Badge>
        </TableCell>
        <TableCell className="max-w-[200px] truncate font-mono text-xs">
          {entry.actor_id || "--"}
        </TableCell>
        <TableCell>{entry.resource || "--"}</TableCell>
        <TableCell className="max-w-[200px] truncate font-mono text-xs">
          {entry.resource_id || "--"}
        </TableCell>
      </TableRow>
      {expanded && hasPayload && (
        <TableRow>
          <TableCell colSpan={6} className="bg-muted/50 p-4">
            <pre className="max-h-80 overflow-auto rounded-md bg-muted p-3 text-xs">
              {JSON.stringify(entry.payload, null, 2)}
            </pre>
          </TableCell>
        </TableRow>
      )}
    </>
  );
}

function AuditLogSkeleton() {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-10" />
            <TableHead><Skeleton className="h-4 w-24" /></TableHead>
            <TableHead><Skeleton className="h-4 w-16" /></TableHead>
            <TableHead><Skeleton className="h-4 w-20" /></TableHead>
            <TableHead><Skeleton className="h-4 w-20" /></TableHead>
            <TableHead><Skeleton className="h-4 w-20" /></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array.from({ length: 8 }, (_, i) => (
            <TableRow key={i}>
              <TableCell className="w-10" />
              {Array.from({ length: 5 }, (_, j) => (
                <TableCell key={j}><Skeleton className="h-4 w-full" /></TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

