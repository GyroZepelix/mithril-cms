import { type ComponentType } from "react";
import {
  StringField,
  TextField,
  RichTextField,
  NumberField,
  BooleanField,
  DateField,
  TimeField,
  EnumField,
  JSONField,
  MediaField,
  RelationField,
} from "@/components/fields";
import type {
  FieldDefinition,
  FieldType,
  FieldComponentProps,
} from "@/lib/types";

/** Maps a schema field type to its corresponding form field component. */
const FIELD_COMPONENT_MAP: Record<FieldType, ComponentType<FieldComponentProps>> = {
  string: StringField,
  text: TextField,
  richtext: RichTextField,
  number: NumberField,
  boolean: BooleanField,
  date: DateField,
  time: TimeField,
  enum: EnumField,
  json: JSONField,
  media: MediaField,
  relation: RelationField,
};

function fieldLabel(field: FieldDefinition): string {
  return field.name
    .replace(/_/g, " ")
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

type ContentFormProps = {
  fields: FieldDefinition[];
  values: Record<string, unknown>;
  errors: Record<string, string>;
  onChange: (name: string, value: unknown) => void;
  disabled?: boolean;
};

/**
 * Renders a dynamic form based on a content type's field definitions.
 * Each field is mapped to its corresponding input component.
 */
export function ContentForm({ fields, values, errors, onChange, disabled }: ContentFormProps) {
  return (
    <div className="space-y-6">
      {fields.map((field) => {
        const Component = FIELD_COMPONENT_MAP[field.type];
        if (!Component) return null;

        return (
          <Component
            key={field.name}
            name={field.name}
            label={fieldLabel(field)}
            value={values[field.name] ?? null}
            onChange={(value) => onChange(field.name, value)}
            error={errors[field.name]}
            field={field}
            disabled={disabled}
          />
        );
      })}
    </div>
  );
}
