/**
 * Shared TypeScript types for the Mithril CMS admin UI.
 * These mirror the backend API response shapes.
 */

// --- Content Type Schema ---

export type FieldType =
  | "string"
  | "text"
  | "richtext"
  | "number"
  | "boolean"
  | "date"
  | "time"
  | "enum"
  | "json"
  | "media"
  | "relation";

export type RelationType = "one" | "many";

export type FieldDefinition = {
  name: string;
  type: FieldType;
  required?: boolean;
  unique?: boolean;
  searchable?: boolean;
  max_length?: number;
  min_length?: number;
  min?: number;
  max?: number;
  regex?: string;
  values?: string[];
  relates_to?: string;
  relation_type?: RelationType;
  media_type?: string;
};

export type ContentTypeSchema = {
  name: string;
  display_name: string;
  public_read: boolean;
  entry_count: number;
  fields: FieldDefinition[];
};

// --- Content Entries ---

/** A content entry is a record with dynamic fields plus system fields. */
export type ContentEntry = {
  id: string;
  status: "draft" | "published";
  created_at: string;
  updated_at: string;
  [key: string]: unknown;
};

export type PaginationMeta = {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
};

export type ContentListResponse = {
  data: ContentEntry[];
  meta: PaginationMeta;
};

// --- API Error ---

export type ValidationDetail = {
  field: string;
  message: string;
};

export type ApiErrorResponse = {
  error: {
    code: string;
    message: string;
    details?: ValidationDetail[];
  };
};

// --- Field Component Props ---

export type FieldComponentProps = {
  name: string;
  label: string;
  value: unknown;
  onChange: (value: unknown) => void;
  error?: string;
  field: FieldDefinition;
  disabled?: boolean;
};
