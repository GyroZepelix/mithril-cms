import { useState, useEffect, useRef } from "react";
import { Textarea } from "@/components/ui/textarea";
import { FieldWrapper } from "./FieldWrapper";
import type { FieldComponentProps } from "@/lib/types";

export function JSONField({ name, label, value, onChange, error, field, disabled }: FieldComponentProps) {
  // Keep raw text in state so the user can type invalid JSON while editing
  const [rawText, setRawText] = useState(() => {
    if (value === null || value === undefined) return "";
    if (typeof value === "string") return value;
    return JSON.stringify(value, null, 2);
  });
  const [parseError, setParseError] = useState<string>();

  // Track the last value we sent to the parent so we can detect external changes
  const lastEmittedValue = useRef<unknown>(value);

  // Sync rawText when the parent updates value externally (e.g. after save)
  useEffect(() => {
    if (value === lastEmittedValue.current) return;
    lastEmittedValue.current = value;
    if (value === null || value === undefined) {
      setRawText("");
    } else if (typeof value === "string") {
      setRawText(value);
    } else {
      setRawText(JSON.stringify(value, null, 2));
    }
    setParseError(undefined);
  }, [value]);

  function handleChange(text: string) {
    setRawText(text);
    if (text.trim() === "") {
      setParseError(undefined);
      lastEmittedValue.current = null;
      onChange(null);
      return;
    }
    try {
      const parsed: unknown = JSON.parse(text);
      setParseError(undefined);
      lastEmittedValue.current = parsed;
      onChange(parsed);
    } catch {
      setParseError("Invalid JSON");
    }
  }

  const displayError = error ?? parseError;

  return (
    <FieldWrapper name={name} label={label} required={field.required} error={displayError}>
      <Textarea
        id={name}
        name={name}
        value={rawText}
        onChange={(e) => handleChange(e.target.value)}
        rows={6}
        className="font-mono text-sm"
        placeholder="{}"
        disabled={disabled}
        aria-invalid={!!displayError}
      />
    </FieldWrapper>
  );
}
