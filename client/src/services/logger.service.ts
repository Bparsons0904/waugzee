/**
 * Client-side logging service that batches logs and sends to VictoriaLogs via backend
 *
 * Usage:
 *   import { logger } from '@services/logger.service';
 *
 *   logger.info('User clicked button', { action: 'sync_start', component: 'SyncButton' });
 *   logger.error('API request failed', { error: { message: err.message, stack: err.stack } });
 *   logger.warn('Deprecated feature used', { component: 'OldComponent' });
 */

import { LOGGER_CONFIG, LOGGING_ENDPOINTS } from "@constants/api.constants";
import type { LogBatchRequest, LogContext, LogEntry, LogLevel, LogMetadata } from "../types/Logger";
import { api } from "./api";
import { env } from "./env.service";

class LoggerService {
  private buffer: LogEntry[] = [];
  private flushTimer: ReturnType<typeof setInterval> | null = null;
  private sessionId: string;
  private isInitialized = false;

  constructor() {
    this.sessionId = this.generateSessionId();
  }

  /**
   * Initialize the logger - call this after auth is ready
   * Sets up flush interval and page unload handler
   */
  initialize(): void {
    if (this.isInitialized) return;

    this.flushTimer = setInterval(() => {
      this.flush();
    }, LOGGER_CONFIG.FLUSH_INTERVAL_MS);

    if (typeof window !== "undefined") {
      window.addEventListener("beforeunload", () => {
        this.flush(true);
      });

      window.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "hidden") {
          this.flush(true);
        }
      });

      window.addEventListener("error", (event) => {
        this.error("Uncaught error", {
          error: {
            message: event.message,
            stack: event.error?.stack,
          },
          context: {
            filename: event.filename,
            lineno: event.lineno,
            colno: event.colno,
          },
        });
      });

      window.addEventListener("unhandledrejection", (event) => {
        const reason = event.reason;
        this.error("Unhandled promise rejection", {
          error: {
            message: reason?.message || String(reason),
            stack: reason?.stack,
          },
        });
      });
    }

    this.isInitialized = true;
  }

  /**
   * Cleanup logger resources
   */
  destroy(): void {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
    this.flush(true);
    this.isInitialized = false;
  }

  /**
   * Log a debug message (development only, not sent to server)
   */
  debug(message: string, context?: LogContext): void {
    if (!env.isProduction) {
      this.logToConsole("debug", message, context);
    }
  }

  /**
   * Log an info message
   */
  info(message: string, context?: LogContext): void {
    this.log("info", message, context);
  }

  /**
   * Log a warning message
   */
  warn(message: string, context?: LogContext): void {
    this.log("warn", message, context);
  }

  /**
   * Log an error message
   */
  error(message: string, context?: LogContext): void {
    this.log("error", message, context);
  }

  /**
   * Core logging method
   */
  private log(level: LogLevel, message: string, context?: LogContext): void {
    // Always output to console in development
    if (!env.isProduction) {
      this.logToConsole(level, message, context);
    }

    const entry: LogEntry = {
      timestamp: new Date().toISOString(),
      level,
      message,
      context,
      metadata: this.getMetadata(),
    };

    this.buffer.push(entry);

    if (this.buffer.length >= LOGGER_CONFIG.BATCH_SIZE) {
      this.flush();
    }

    if (this.buffer.length > LOGGER_CONFIG.MAX_BUFFER_SIZE) {
      this.buffer = this.buffer.slice(-LOGGER_CONFIG.MAX_BUFFER_SIZE);
    }
  }

  /**
   * Output log to browser console
   */
  private logToConsole(level: LogLevel | "debug", message: string, context?: LogContext): void {
    const prefix = `[${level.toUpperCase()}]`;
    const args: unknown[] = [prefix, message];

    if (context) {
      args.push(context);
    }

    switch (level) {
      case "debug":
        console.debug(...args);
        break;
      case "info":
        console.info(...args);
        break;
      case "warn":
        console.warn(...args);
        break;
      case "error":
        console.error(...args);
        break;
    }
  }

  /**
   * Flush buffered logs to the server
   */
  private flush(sync = false): void {
    if (this.buffer.length === 0) return;

    const logs = [...this.buffer];
    this.buffer = [];

    const payload: LogBatchRequest = {
      logs,
      sessionId: this.sessionId,
    };

    if (sync && typeof navigator !== "undefined" && navigator.sendBeacon) {
      const blob = new Blob([JSON.stringify(payload)], {
        type: "application/json",
      });
      navigator.sendBeacon(`${this.getApiBaseUrl()}${LOGGING_ENDPOINTS.BATCH}`, blob);
    } else {
      api.post(LOGGING_ENDPOINTS.BATCH, payload).catch((err) => {
        console.error("Failed to send logs:", err);
        this.buffer = [...logs, ...this.buffer].slice(-LOGGER_CONFIG.MAX_BUFFER_SIZE);
      });
    }
  }

  /**
   * Get current page metadata
   */
  private getMetadata(): LogMetadata {
    if (typeof window === "undefined") {
      return {
        userAgent: "",
        url: "",
        referrer: "",
        viewport: { width: 0, height: 0 },
      };
    }

    return {
      userAgent: navigator.userAgent,
      url: window.location.href,
      referrer: document.referrer,
      viewport: {
        width: window.innerWidth,
        height: window.innerHeight,
      },
    };
  }

  /**
   * Generate a unique session ID
   */
  private generateSessionId(): string {
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substring(2, 9);
    return `${timestamp}-${random}`;
  }

  /**
   * Get the API base URL for sendBeacon
   */
  private getApiBaseUrl(): string {
    const apiUrl = import.meta.env.VITE_API_URL || "http://localhost:8288";
    return `${apiUrl}/api`;
  }

  /**
   * Get current buffer size (for debugging)
   */
  getBufferSize(): number {
    return this.buffer.length;
  }

  /**
   * Force flush (for testing/debugging)
   */
  forceFlush(): void {
    this.flush();
  }
}

export const logger = new LoggerService();
