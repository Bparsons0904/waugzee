// Claude why do we need this? I don't think we are using it and we should let tanStack handle retrys
import { RETRY_CONFIG } from "@constants/api.constants";
import { ApiClientError, NetworkError } from "@services/api";

/**
 * Exponential backoff retry utility for handling network errors
 */

export interface RetryConfig {
  maxAttempts?: number;
  baseDelayMs?: number;
  maxDelayMs?: number;
  exponentialBase?: number;
  shouldRetry?: (error: Error) => boolean;
}

/**
 * Default retry condition - retry on network errors and 5xx server errors
 */
const defaultShouldRetry = (error: Error): boolean => {
  // Always retry network errors
  if (error instanceof NetworkError) {
    return true;
  }

  // Retry server errors (5xx)
  if (error instanceof ApiClientError) {
    return error.status >= 500 && error.status < 600;
  }

  // Don't retry other errors (4xx client errors, etc.)
  return false;
};

/**
 * Calculate exponential backoff delay with jitter
 */
const calculateDelay = (
  attempt: number,
  baseDelay: number,
  maxDelay: number,
  exponentialBase: number,
): number => {
  const exponentialDelay = baseDelay * Math.pow(exponentialBase, attempt - 1);
  const delayWithCap = Math.min(exponentialDelay, maxDelay);

  // Add jitter (±25% randomness)
  const jitterFactor = 0.25;
  const jitter = 1 + (Math.random() * 2 - 1) * jitterFactor;

  return Math.floor(delayWithCap * jitter);
};

/**
 * Sleep utility for delays
 */
const sleep = (ms: number): Promise<void> => {
  return new Promise((resolve) => setTimeout(resolve, ms));
};

/**
 * Retry a function with exponential backoff
 */
export async function retryWithExponentialBackoff<T>(
  fn: () => Promise<T>,
  config: RetryConfig = {},
): Promise<T> {
  const {
    maxAttempts = RETRY_CONFIG.MAX_ATTEMPTS,
    baseDelayMs = RETRY_CONFIG.BASE_DELAY_MS,
    maxDelayMs = RETRY_CONFIG.MAX_DELAY_MS,
    exponentialBase = RETRY_CONFIG.EXPONENTIAL_BASE,
    shouldRetry = defaultShouldRetry,
  } = config;

  let lastError: Error;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;

      // Don't retry on the last attempt or if error shouldn't be retried
      if (attempt === maxAttempts || !shouldRetry(lastError)) {
        throw lastError;
      }

      // Calculate and apply delay
      const delay = calculateDelay(
        attempt,
        baseDelayMs,
        maxDelayMs,
        exponentialBase,
      );

      console.debug(
        `Retry attempt ${attempt}/${maxAttempts} failed, retrying in ${delay}ms:`,
        {
          error: lastError.message,
          attempt,
          delay,
        },
      );

      await sleep(delay);
    }
  }

  throw lastError!;
}

/**
 * Specific retry configuration for auth-related requests
 */
export const authRetryConfig: RetryConfig = {
  maxAttempts: 2, // More conservative for auth
  shouldRetry: (error: Error) => {
    // Only retry network errors for auth requests
    return error instanceof NetworkError;
  },
};

/**
 * Specific retry configuration for data requests
 */
export const dataRetryConfig: RetryConfig = {
  maxAttempts: RETRY_CONFIG.MAX_ATTEMPTS,
  shouldRetry: defaultShouldRetry,
};

