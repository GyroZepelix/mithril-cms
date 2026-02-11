import { useParams, Link, Navigate } from "react-router";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

const VALID_TYPE_PATTERN = /^[a-z][a-z0-9_]*$/;

export function ContentListPage() {
  const { type } = useParams<{ type: string }>();

  if (!type || !VALID_TYPE_PATTERN.test(type)) {
    return <Navigate to="/admin" replace />;
  }

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold capitalize">
          {type.replace(/_/g, " ")}
        </h1>
        <Button asChild>
          <Link to={`/admin/content/${type}/new`}>
            <Plus className="h-4 w-4" />
            New Entry
          </Link>
        </Button>
      </div>
      <p className="text-muted-foreground">
        Content list for &ldquo;{type}&rdquo; will be implemented here.
      </p>
    </div>
  );
}
