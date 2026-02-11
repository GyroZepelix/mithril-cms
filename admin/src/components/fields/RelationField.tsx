import { Input } from "@/components/ui/input";
import { FieldWrapper } from "./FieldWrapper";
import type { FieldComponentProps } from "@/lib/types";

/** Placeholder relation field -- currently renders a UUID text input. Will be enhanced with a search/select in a future task. */
export function RelationField({ name, label, value, onChange, error, field, disabled }: FieldComponentProps) {
  const strValue = typeof value === "string" ? value : "";

  return (
    <FieldWrapper name={name} label={label} required={field.required} error={error}>
      <Input
        id={name}
        name={name}
        value={strValue}
        onChange={(e) => onChange(e.target.value || null)}
        placeholder={`${field.relates_to ?? "Related"} UUID`}
        disabled={disabled}
        aria-invalid={!!error}
      />
      <p className="text-xs text-muted-foreground">
        Relates to: {field.relates_to ?? "unknown"} ({field.relation_type ?? "one"})
      </p>
    </FieldWrapper>
  );
}
