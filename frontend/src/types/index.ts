// src/types/index.ts

export interface Session {
  id: string;
  user_id: string;
  table_name: string;
  created_at: string;
  last_used: string;
}

export interface QueryResponse {
  data: Record<string, any>[];
  runtime: string;
  query: string;
  row_count: number;
  session_id: string;
  cached: boolean;
}

export interface NaturalResponse {
  natural_query: string;
  generated_sql: string;
  explanation: string;
  runtime: string;
  data: Record<string, any>[];
  row_count: number;
  session_id: string;
  cached: boolean;
}

export interface UploadResponse {
  message: string;
  filename: string;
  rows: number;
  columns: string[];
  table: string;
  session_id: string;
}

export interface QueryHistoryItem {
  id: string;
  type: 'sql' | 'natural';
  query: string;
  generated_sql?: string;
  row_count: number;
  runtime: string;
  timestamp: Date;
}

export interface SessionResponse {
  session_id: string;
  message: string;
}

export interface SchemaResponse {
  schema: string;
  session_id: string;
}