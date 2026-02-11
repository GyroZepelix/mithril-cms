import { Input } from "@/components/ui/input";
import { FieldWrapper } from "./FieldWrapper";
import type { FieldComponentProps } from "@/lib/types";

/** Placeholder media field -- currently renders a UUID text input. Will be enhanced with file upload in a future task. */
export function MediaField({ name, label, value, onChange, error, field, disabled }: FieldComponentProps) {
  const strValue = typeof value === "string" ? value : "";

  return (
    <FieldWrapper name={name} label={label} required={field.required} error={error}>
      <Input
        id={name}
        name={name}
        value={strValue}
        onChange={(e) => onChange(e.target.value || null)}
        placeholder="Media UUID"
        disabled={disabled}
        aria-invalid={!!error}
      />
      <p className="text-xs text-muted-foreground">
        Enter a media ID. File picker coming soon.
      </p>
    </FieldWrapper>
  );
}
