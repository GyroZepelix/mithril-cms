import { useState, useEffect, useRef } from "react";
import { useParams, Link, Navigate, useNavigate, useSearchParams } from "react-router";
import { Plus, ArrowUpDown, ArrowUp, ArrowDown, Search, Loader2 } from "lucide-react";
import { api } from "@/lib/api";
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
import type {
  ContentTypeSchema,
  ContentEntry,
  PaginationMeta,
  FieldDefinition,
} from "@/lib/types";

const VALID_TYPE_PATTERN = /^[a-z][a-z0-9_]*$/;
const DEFAULT_PER_PAGE = 20;

/** Maximum number of user-defined columns to show in the table. */
const MAX_VISIBLE_COLUMNS = 5;

type SortOrder = "asc" | "desc";

function formatCellValue(value: unknown, field: FieldDefinition): string {
  if (value === null || value === undefined) return "--";
  if (field.type === "boolean") return value ? "Yes" : "No";
  if (field.type === "json") return typeof value === "string" ? value : JSON.stringify(value);
  if (typeof value === "string" && value.length > 80) return value.slice(0, 80) + "...";
  return String(value);
}

function formatDate(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleDateString(undefined, {
      year: "numeric",
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

export function ContentListPage() {
  const { type } = useParams<{ type: string }>();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();

  const [schema, setSchema] = useState<ContentTypeSchema | null>(null);
  const [entries, setEntries] = useState<ContentEntry[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Read query params with defaults
  const page = Number(searchParams.get("page")) || 1;
  const perPage = Math.min(Math.max(Number(searchParams.get("per_page")) || DEFAULT_PER_PAGE, 1), 100);
  const sort = searchParams.get("sort") || "created_at";
  const order = (searchParams.get("order") || "desc") as SortOrder;
  const query = searchParams.get("q") || "";

  // Search input is debounced, so we track the input value separately
  const [searchInput, setSearchInput] = useState(query);

  const isInitialMount = useRef(true);

  // Fetch schema once
  useEffect(() => {
    let cancelled = false;

    async function fetchSchema() {
      try {
        const data = await api.get<ContentTypeSchema>(`/admin/api/content-types/${type}`);
        if (!cancelled) setSchema(data);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : "Failed to load content type");
      }
    }

    fetchSchema();
    return () => { cancelled = true; };
  }, [type]);

  // Fetch entries when params change
  useEffect(() => {
    let cancelled = false;

    async function fetchEntries() {
      setLoading(true);
      setError(null);
      try {
        const params = new URLSearchParams({
          page: String(page),
          per_page: String(perPage),
          sort,
          order,
        });
        if (query) params.set("q", query);

        const result = await api.getWithMeta<ContentEntry[], PaginationMeta>(
          `/admin/api/content/${type}?${params.toString()}`
        );
        if (!cancelled) {
          setEntries(result.data);
          setMeta(result.meta);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load entries");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchEntries();
    return () => { cancelled = true; };
  }, [type, page, perPage, sort, order, query]);

  // Debounce search input (skip initial mount to avoid resetting page to 1)
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }
    const timer = setTimeout(() => {
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev);
        if (searchInput) {
          next.set("q", searchInput);
        } else {
          next.delete("q");
        }
        next.set("page", "1"); // Reset to first page on search
        return next;
      });
    }, 300);
    return () => clearTimeout(timer);
  }, [searchInput, setSearchParams]);

  if (!type || !VALID_TYPE_PATTERN.test(type)) {
    return <Navigate to="/admin" replace />;
  }

  function handleSort(field: string) {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev);
      if (sort === field) {
        next.set("order", order === "asc" ? "desc" : "asc");
      } else {
        next.set("sort", field);
        next.set("order", "asc");
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

  function handleRowClick(entryId: string) {
    navigate(`/admin/content/${type}/${entryId}`);
  }

  function SortIcon({ field }: { field: string }) {
    if (sort !== field) return <ArrowUpDown className="ml-1 h-3 w-3 opacity-50" />;
    return order === "asc"
      ? <ArrowUp className="ml-1 h-3 w-3" />
      : <ArrowDown className="ml-1 h-3 w-3" />;
  }

  // Determine visible columns from schema fields
  const visibleFields = schema?.fields.slice(0, MAX_VISIBLE_COLUMNS) ?? [];

  const displayName = schema?.display_name ?? type.replace(/_/g, " ");

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">{displayName}</h1>
        <Button asChild>
          <Link to={`/admin/content/${type}/new`}>
            <Plus className="h-4 w-4" />
            New Entry
          </Link>
        </Button>
      </div>

      {/* Search */}
      <div className="mb-4 flex items-center gap-4">
        <div className="relative max-w-sm flex-1">
          <Search className="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search entries..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            className="pl-9"
          />
        </div>
        {meta && (
          <p className="text-sm text-muted-foreground">
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
      {loading && !entries.length ? (
        <TableSkeleton columns={visibleFields.length + 2} />
      ) : entries.length === 0 && !loading ? (
        <div className="flex flex-col items-center justify-center rounded-md border border-dashed py-16">
          <p className="mb-4 text-muted-foreground">
            {query ? "No entries match your search." : "No entries yet."}
          </p>
          {!query && (
            <Button asChild>
              <Link to={`/admin/content/${type}/new`}>
                <Plus className="h-4 w-4" />
                Create first entry
              </Link>
            </Button>
          )}
        </div>
      ) : (
        <>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  {/* Status column */}
                  <TableHead className="w-[100px]">Status</TableHead>

                  {/* Dynamic columns from schema */}
                  {visibleFields.map((field) => (
                    <TableHead key={field.name}>
                      <button
                        type="button"
                        className="inline-flex items-center font-medium hover:text-foreground"
                        onClick={() => handleSort(field.name)}
                      >
                        {field.name.replace(/_/g, " ")}
                        <SortIcon field={field.name} />
                      </button>
                    </TableHead>
                  ))}

                  {/* Updated at column */}
                  <TableHead>
                    <button
                      type="button"
                      className="inline-flex items-center font-medium hover:text-foreground"
                      onClick={() => handleSort("updated_at")}
                    >
                      Updated
                      <SortIcon field="updated_at" />
                    </button>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {loading && (
                  <TableRow>
                    <TableCell colSpan={visibleFields.length + 2} className="text-center">
                      <Loader2 className="mx-auto h-4 w-4 animate-spin" />
                    </TableCell>
                  </TableRow>
                )}
                {!loading && entries.map((entry) => (
                  <TableRow
                    key={entry.id}
                    className="cursor-pointer"
                    onClick={() => handleRowClick(entry.id)}
                  >
                    <TableCell>
                      <StatusBadge status={entry.status} />
                    </TableCell>
                    {visibleFields.map((field) => (
                      <TableCell key={field.name} className="max-w-[250px] truncate">
                        {formatCellValue(entry[field.name], field)}
                      </TableCell>
                    ))}
                    <TableCell className="text-muted-foreground">
                      {formatDate(entry.updated_at)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {/* Pagination */}
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

function StatusBadge({ status }: { status: string }) {
  if (status === "published") {
    return <Badge className="bg-green-600 hover:bg-green-600/80">Published</Badge>;
  }
  return <Badge variant="secondary">Draft</Badge>;
}

function TableSkeleton({ columns }: { columns: number }) {
  return (
    <div className="rounded-md border">
      <Table>
        <TableHeader>
          <TableRow>
            {Array.from({ length: columns }, (_, i) => (
              <TableHead key={i}>
                <Skeleton className="h-4 w-20" />
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {Array.from({ length: 5 }, (_, row) => (
            <TableRow key={row}>
              {Array.from({ length: columns }, (_, col) => (
                <TableCell key={col}>
                  <Skeleton className="h-4 w-full" />
                </TableCell>
              ))}
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function Pagination({
  page,
  totalPages,
  onPageChange,
}: {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
}) {
  return (
    <div className="mt-4 flex items-center justify-between">
      <p className="text-sm text-muted-foreground">
        Page {page} of {totalPages}
      </p>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          Previous
        </Button>
        <Button
          variant="outline"
          size="sm"
          disabled={page >= totalPages}
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </Button>
      </div>
    </div>
  );
}
