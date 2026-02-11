import { Input } from "@/components/ui/input";
import { FieldWrapper } from "./FieldWrapper";
import type { FieldComponentProps } from "@/lib/types";

export function TimeField({ name, label, value, onChange, error, field, disabled }: FieldComponentProps) {
  const strValue = typeof value === "string" ? value : "";

  return (
    <FieldWrapper name={name} label={label} required={field.required} error={error}>
      <Input
        id={name}
        name={name}
        type="time"
        value={strValue}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        aria-invalid={!!error}
      />
    </FieldWrapper>
  );
}
