import { Input } from "@/components/ui/input";
import { FieldWrapper } from "./FieldWrapper";
import type { FieldComponentProps } from "@/lib/types";

export function StringField({ name, label, value, onChange, error, field, disabled }: FieldComponentProps) {
  const strValue = typeof value === "string" ? value : "";

  return (
    <FieldWrapper name={name} label={label} required={field.required} error={error}>
      <div className="relative">
        <Input
          id={name}
          name={name}
          value={strValue}
          onChange={(e) => onChange(e.target.value)}
          maxLength={field.max_length}
          disabled={disabled}
          aria-invalid={!!error}
          aria-describedby={error ? `${name}-error` : undefined}
        />
        {field.max_length && (
          <span className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-muted-foreground">
            {strValue.length}/{field.max_length}
          </span>
        )}
      </div>
    </FieldWrapper>
  );
}
