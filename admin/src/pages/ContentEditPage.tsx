import { useState, useEffect } from "react";
import { useParams, Navigate, useNavigate, Link } from "react-router";
import { ArrowLeft, Loader2, Save, Globe } from "lucide-react";
import { api, ApiRequestError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { ContentForm } from "@/components/ContentForm";
import type {
  ContentTypeSchema,
  ContentEntry,
  ApiErrorResponse,
  ValidationDetail,
} from "@/lib/types";

const VALID_TYPE_PATTERN = /^[a-z][a-z0-9_]*$/;

/** Extract field-level validation errors from an API error response. */
function parseFieldErrors(body: unknown): Record<string, string> {
  const errors: Record<string, string> = {};
  if (!body || typeof body !== "object") return errors;

  const apiError = body as ApiErrorResponse;
  const details = apiError.error?.details;
  if (!Array.isArray(details)) return errors;

  for (const detail of details as ValidationDetail[]) {
    if (detail.field && detail.message) {
      errors[detail.field] = detail.message;
    }
  }
  return errors;
}

export function ContentEditPage() {
  const { type, id } = useParams<{ type: string; id: string }>();
  const navigate = useNavigate();
  const isNew = !id;

  const [schema, setSchema] = useState<ContentTypeSchema | null>(null);
  const [values, setValues] = useState<Record<string, unknown>>({});
  const [entry, setEntry] = useState<ContentEntry | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [globalError, setGlobalError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);

  // Fetch schema (always) and entry (in edit mode)
  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      setGlobalError(null);

      try {
        const schemaData = await api.get<ContentTypeSchema>(`/admin/api/content-types/${type}`);
        if (cancelled) return;
        setSchema(schemaData);

        if (id) {
          const entryData = await api.get<ContentEntry>(`/admin/api/content/${type}/${id}`);
          if (cancelled) return;
          setEntry(entryData);

          // Initialize form values from entry
          const initial: Record<string, unknown> = {};
          for (const field of schemaData.fields) {
            initial[field.name] = entryData[field.name] ?? null;
          }
          setValues(initial);
        } else {
          // Initialize empty form for new entry
          const initial: Record<string, unknown> = {};
          for (const field of schemaData.fields) {
            initial[field.name] = field.type === "boolean" ? false : null;
          }
          setValues(initial);
        }
      } catch (err) {
        if (!cancelled) {
          setGlobalError(err instanceof Error ? err.message : "Failed to load content");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => { cancelled = true; };
  }, [type, id]);

  if (!type || !VALID_TYPE_PATTERN.test(type)) {
    return <Navigate to="/admin" replace />;
  }

  function handleFieldChange(name: string, value: unknown) {
    setValues((prev) => ({ ...prev, [name]: value }));
    // Clear field error when user modifies the field
    if (fieldErrors[name]) {
      setFieldErrors((prev) => {
        const next = { ...prev };
        delete next[name];
        return next;
      });
    }
  }

  async function handleSave() {
    if (!schema) return;

    setSaving(true);
    setFieldErrors({});
    setGlobalError(null);

    try {
      // Build payload: always include required fields so the server can validate;
      // only strip truly unset optional fields.
      const payload: Record<string, unknown> = {};
      for (const field of schema.fields) {
        const val = values[field.name];
        if (field.required || (val !== null && val !== undefined && val !== "")) {
          payload[field.name] = val;
        }
      }

      if (isNew) {
        const created = await api.post<ContentEntry>(`/admin/api/content/${type}`, payload);
        // Navigate to the edit page for the newly created entry
        navigate(`/admin/content/${type}/${created.id}`, { replace: true });
      } else {
        const updated = await api.put<ContentEntry>(`/admin/api/content/${type}/${id}`, payload);
        setEntry(updated);
      }
    } catch (err) {
      if (err instanceof ApiRequestError) {
        const errors = parseFieldErrors(err.body);
        if (Object.keys(errors).length > 0) {
          setFieldErrors(errors);
        } else {
          setGlobalError(err.message);
        }
      } else {
        setGlobalError(err instanceof Error ? err.message : "Save failed");
      }
    } finally {
      setSaving(false);
    }
  }

  async function handlePublish() {
    if (!id) return;

    setPublishing(true);
    setGlobalError(null);

    try {
      const updated = await api.post<ContentEntry>(`/admin/api/content/${type}/${id}/publish`);
      setEntry(updated);
    } catch (err) {
      if (err instanceof ApiRequestError) {
        const errors = parseFieldErrors(err.body);
        if (Object.keys(errors).length > 0) {
          setFieldErrors(errors);
        } else {
          setGlobalError(err.message);
        }
      } else {
        setGlobalError(err instanceof Error ? err.message : "Publish failed");
      }
    } finally {
      setPublishing(false);
    }
  }

  const displayName = schema?.display_name ?? type.replace(/_/g, " ");
  const isBusy = saving || publishing;

  // Loading state
  if (loading) {
    return (
      <div>
        <div className="mb-6 flex items-center gap-4">
          <Skeleton className="h-9 w-24" />
          <Skeleton className="h-8 w-48" />
        </div>
        <div className="max-w-2xl space-y-6">
          {Array.from({ length: 4 }, (_, i) => (
            <div key={i} className="space-y-2">
              <Skeleton className="h-4 w-24" />
              <Skeleton className="h-9 w-full" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  // Error loading content type or entry
  if (globalError && !schema) {
    return (
      <div>
        <BackLink type={type} />
        <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          {globalError}
        </div>
      </div>
    );
  }

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <BackLink type={type} />
          <h1 className="text-2xl font-bold">
            {isNew ? "New" : "Edit"} {displayName}
          </h1>
          {entry && (
            <Badge
              variant={entry.status === "published" ? "default" : "secondary"}
              className={entry.status === "published" ? "bg-green-600 hover:bg-green-600/80" : ""}
            >
              {entry.status === "published" ? "Published" : "Draft"}
            </Badge>
          )}
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={handleSave}
            disabled={isBusy}
          >
            {saving ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Save className="h-4 w-4" />
            )}
            {isNew ? "Create" : "Save Draft"}
          </Button>

          {!isNew && (
            <Button
              onClick={handlePublish}
              disabled={isBusy}
            >
              {publishing ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Globe className="h-4 w-4" />
              )}
              Publish
            </Button>
          )}
        </div>
      </div>

      <Separator className="mb-6" />

      {/* Global error */}
      {globalError && (
        <div className="mb-6 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          {globalError}
        </div>
      )}

      {/* Form */}
      {schema && (
        <div className="max-w-2xl">
          <ContentForm
            fields={schema.fields}
            values={values}
            errors={fieldErrors}
            onChange={handleFieldChange}
            disabled={isBusy}
          />
        </div>
      )}

      {/* Bottom action bar for long forms */}
      {schema && schema.fields.length > 4 && (
        <>
          <Separator className="my-6" />
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={handleSave}
              disabled={isBusy}
            >
              {saving ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Save className="h-4 w-4" />
              )}
              {isNew ? "Create" : "Save Draft"}
            </Button>
            {!isNew && (
              <Button
                onClick={handlePublish}
                disabled={isBusy}
              >
                {publishing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Globe className="h-4 w-4" />
                )}
                Publish
              </Button>
            )}
          </div>
        </>
      )}
    </div>
  );
}

function BackLink({ type }: { type: string }) {
  return (
    <Button variant="ghost" size="icon" asChild>
      <Link to={`/admin/content/${type}`} aria-label="Back to list">
        <ArrowLeft className="h-4 w-4" />
      </Link>
    </Button>
  );
}
