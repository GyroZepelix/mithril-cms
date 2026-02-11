import { useState, useEffect, useCallback, useRef } from "react";
import { FileIcon, ImageIcon, Loader2 } from "lucide-react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { MediaUploader } from "@/components/media/MediaUploader";
import { MediaDetail } from "@/components/media/MediaDetail";
import { formatBytes, formatDateShort, isImageMime } from "@/lib/format";
import type { MediaRecord, PaginationMeta } from "@/lib/types";

const PER_PAGE = 20;

function getThumbnailUrl(media: MediaRecord): string | null {
  if (!isImageMime(media.mime_type)) return null;
  // Prefer the small variant for thumbnails, fall back to original
  if (media.variants.sm) {
    return `/media/${media.filename}?v=sm`;
  }
  return `/media/${media.filename}`;
}

export function MediaPage() {
  const [media, setMedia] = useState<MediaRecord[]>([]);
  const [meta, setMeta] = useState<PaginationMeta | null>(null);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [selectedMedia, setSelectedMedia] = useState<MediaRecord | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);

  const fetchIdRef = useRef(0);

  const fetchMedia = useCallback(async (targetPage: number) => {
    const id = ++fetchIdRef.current;
    setLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams({
        page: String(targetPage),
        per_page: String(PER_PAGE),
      });
      const result = await api.getWithMeta<MediaRecord[], PaginationMeta>(
        `/admin/api/media?${params.toString()}`,
      );
      if (id !== fetchIdRef.current) return; // stale request
      setMedia(result.data);
      setMeta(result.meta);
    } catch (err) {
      if (id !== fetchIdRef.current) return; // stale request
      setError(err instanceof Error ? err.message : "Failed to load media");
    } finally {
      if (id === fetchIdRef.current) setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchMedia(page);
  }, [page, fetchMedia]);

  function handleUploadComplete() {
    // Refresh current page after upload
    fetchMedia(page);
  }

  function handleCardClick(record: MediaRecord) {
    setSelectedMedia(record);
    setDetailOpen(true);
  }

  function handleCardKeyDown(e: React.KeyboardEvent, record: MediaRecord) {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      handleCardClick(record);
    }
  }

  function handleDeleted() {
    // If we deleted the last item on a page > 1, go back one page
    if (media.length === 1 && page > 1) {
      setPage(page - 1);
    } else {
      fetchMedia(page);
    }
  }

  return (
    <div>
      {/* Header */}
      <h1 className="mb-6 text-2xl font-bold">Media Library</h1>

      {/* Upload area */}
      <div className="mb-6">
        <MediaUploader onUploadComplete={handleUploadComplete} />
      </div>

      {/* Error */}
      {error && (
        <div className="mb-4 rounded-md border border-destructive/50 bg-destructive/10 p-4 text-sm text-destructive">
          {error}
        </div>
      )}

      {/* Media count */}
      {meta && (
        <p className="mb-4 text-sm text-muted-foreground">
          {meta.total} {meta.total === 1 ? "file" : "files"}
        </p>
      )}

      {/* Grid */}
      {loading && media.length === 0 ? (
        <MediaGridSkeleton />
      ) : media.length === 0 && !loading ? (
        <div className="flex flex-col items-center justify-center rounded-md border border-dashed py-16">
          <ImageIcon className="mb-4 h-12 w-12 text-muted-foreground" />
          <p className="text-muted-foreground">
            No media files yet. Upload one above.
          </p>
        </div>
      ) : (
        <>
          {loading && (
            <div className="mb-4 flex items-center justify-center">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
            </div>
          )}

          <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
            {media.map((record) => (
              <MediaCard
                key={record.id}
                media={record}
                onClick={() => handleCardClick(record)}
                onKeyDown={(e) => handleCardKeyDown(e, record)}
              />
            ))}
          </div>

          {/* Pagination */}
          {meta && meta.total_pages > 1 && (
            <div className="mt-6 flex items-center justify-between">
              <p className="text-sm text-muted-foreground">
                Page {meta.page} of {meta.total_pages}
              </p>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page <= 1}
                  onClick={() => setPage(page - 1)}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={page >= meta.total_pages}
                  onClick={() => setPage(page + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Detail dialog */}
      {selectedMedia && (
        <MediaDetail
          media={selectedMedia}
          open={detailOpen}
          onOpenChange={setDetailOpen}
          onDeleted={handleDeleted}
        />
      )}
    </div>
  );
}

function MediaCard({
  media,
  onClick,
  onKeyDown,
}: {
  media: MediaRecord;
  onClick: () => void;
  onKeyDown: (e: React.KeyboardEvent) => void;
}) {
  const thumbnailUrl = getThumbnailUrl(media);

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onClick}
      onKeyDown={onKeyDown}
      className="group cursor-pointer overflow-hidden rounded-lg border bg-card transition-colors hover:border-primary/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
    >
      {/* Thumbnail area */}
      <div className="flex aspect-square items-center justify-center overflow-hidden bg-muted/50">
        {thumbnailUrl ? (
          <img
            src={thumbnailUrl}
            alt={media.original_name}
            className="h-full w-full object-cover"
            loading="lazy"
          />
        ) : (
          <FileIcon className="h-12 w-12 text-muted-foreground" />
        )}
      </div>

      {/* Info */}
      <div className="p-2">
        <p className="truncate text-xs font-medium" title={media.original_name}>
          {media.original_name}
        </p>
        <p className="text-xs text-muted-foreground">
          {formatBytes(media.size)} &middot; {formatDateShort(media.created_at)}
        </p>
      </div>
    </div>
  );
}

function MediaGridSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
      {Array.from({ length: 10 }, (_, i) => (
        <div key={i} className="overflow-hidden rounded-lg border">
          <Skeleton className="aspect-square w-full" />
          <div className="space-y-1 p-2">
            <Skeleton className="h-3 w-3/4" />
            <Skeleton className="h-3 w-1/2" />
          </div>
        </div>
      ))}
    </div>
  );
}
