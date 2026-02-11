import { FieldWrapper } from "./FieldWrapper";
import { cn } from "@/lib/utils";
import type { FieldComponentProps } from "@/lib/types";

export function EnumField({ name, label, value, onChange, error, field, disabled }: FieldComponentProps) {
  const strValue = typeof value === "string" ? value : "";
  const values = field.values ?? [];

  return (
    <FieldWrapper name={name} label={label} required={field.required} error={error}>
      <select
        id={name}
        name={name}
        value={strValue}
        onChange={(e) => onChange(e.target.value || null)}
        disabled={disabled}
        aria-invalid={!!error}
        className={cn(
          "flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors",
          "focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring",
          "disabled:cursor-not-allowed disabled:opacity-50",
        )}
      >
        <option value="">Select...</option>
        {values.map((v) => (
          <option key={v} value={v}>
            {v}
          </option>
        ))}
      </select>
    </FieldWrapper>
  );
}
