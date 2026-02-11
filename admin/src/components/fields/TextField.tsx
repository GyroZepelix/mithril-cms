import { Textarea } from "@/components/ui/textarea";
import { FieldWrapper } from "./FieldWrapper";
import type { FieldComponentProps } from "@/lib/types";

export function TextField({ name, label, value, onChange, error, field, disabled }: FieldComponentProps) {
  const strValue = typeof value === "string" ? value : "";

  return (
    <FieldWrapper name={name} label={label} required={field.required} error={error}>
      <Textarea
        id={name}
        name={name}
        value={strValue}
        onChange={(e) => onChange(e.target.value)}
        rows={4}
        maxLength={field.max_length}
        disabled={disabled}
        aria-invalid={!!error}
      />
    </FieldWrapper>
  );
}
