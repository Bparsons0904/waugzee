/**
 * Logger types for client-side logging to VictoriaLogs
 */

export type LogLevel = "info" | "warn" | "error";

export interface LogContext {
  action?: string;
  releaseId?: string;
  stylusId?: string;
  component?: string;
  duration?: number;
  error?: {
    message: string;
    stack?: string;
    code?: string;
  };
  [key: string]: unknown;
}

export interface LogMetadata {
  userAgent: string;
  url: string;
  referrer: string;
  viewport: {
    width: number;
    height: number;
  };
}

export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
  context?: LogContext;
  metadata: LogMetadata;
}

export interface LogBatchRequest {
  logs: LogEntry[];
  sessionId: string;
}

export interface LogBatchResponse {
  success: boolean;
  processed: number;
}
