import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import type { FieldComponentProps } from "@/lib/types";

export function BooleanField({ name, label, value, onChange, error, disabled }: FieldComponentProps) {
  const checked = value === true;

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-3">
        <Switch
          id={name}
          checked={checked}
          onCheckedChange={(val) => onChange(val)}
          disabled={disabled}
          aria-invalid={!!error}
        />
        <Label htmlFor={name}>{label}</Label>
      </div>
      {error && (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
