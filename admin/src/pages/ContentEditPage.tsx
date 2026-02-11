import { useParams, Navigate } from "react-router";

const VALID_TYPE_PATTERN = /^[a-z][a-z0-9_]*$/;

export function ContentEditPage() {
  const { type, id } = useParams<{ type: string; id: string }>();
  const isNew = !id;

  if (!type || !VALID_TYPE_PATTERN.test(type)) {
    return <Navigate to="/admin" replace />;
  }

  return (
    <div>
      <h1 className="mb-6 text-2xl font-bold capitalize">
        {isNew ? "New" : "Edit"} {type.replace(/_/g, " ")}
      </h1>
      <p className="text-muted-foreground">
        Content editor for &ldquo;{type}&rdquo; will be implemented here.
      </p>
    </div>
  );
}
