import type { ReactNode } from "react";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

type FieldWrapperProps = {
  name: string;
  label: string;
  required?: boolean;
  error?: string;
  children: ReactNode;
  className?: string;
};

/**
 * Shared wrapper for all field components.
 * Renders a label, the field input, and an optional error message.
 */
export function FieldWrapper({ name, label, required, error, children, className }: FieldWrapperProps) {
  return (
    <div className={cn("space-y-2", className)}>
      <Label htmlFor={name}>
        {label}
        {required && <span className="ml-1 text-destructive">*</span>}
      </Label>
      {children}
      {error && (
        <p id={`${name}-error`} className="text-sm text-destructive" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}
