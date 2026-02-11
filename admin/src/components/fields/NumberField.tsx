import { Input } from "@/components/ui/input";
import { FieldWrapper } from "./FieldWrapper";
import type { FieldComponentProps } from "@/lib/types";

export function NumberField({ name, label, value, onChange, error, field, disabled }: FieldComponentProps) {
  const numValue = typeof value === "number" ? String(value) : value === null || value === undefined ? "" : String(value);

  return (
    <FieldWrapper name={name} label={label} required={field.required} error={error}>
      <Input
        id={name}
        name={name}
        type="number"
        value={numValue}
        onChange={(e) => {
          const raw = e.target.value;
          if (raw === "") {
            onChange(null);
          } else {
            const parsed = Number(raw);
            if (!Number.isNaN(parsed)) {
              onChange(parsed);
            }
          }
        }}
        min={field.min}
        max={field.max}
        step={field.type === "number" ? "any" : undefined}
        disabled={disabled}
        aria-invalid={!!error}
      />
    </FieldWrapper>
  );
}
