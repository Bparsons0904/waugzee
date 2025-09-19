export interface DiscogsApiRequest {
  requestId: string;
  url: string;
  method: string;
  headers: Record<string, string>;
}

export interface DiscogsApiResponse {
  requestId: string;
  status: number;
  headers: Record<string, string>;
  body: unknown;
  error?: string;
}

export interface RateLimitInfo {
  remaining: number;
  limit: number;
  window: number;
  resetTime: Date;
}

export class DiscogsApiService {
  /**
   * Makes an HTTP request to the Discogs API
   */
  async makeRequest(request: DiscogsApiRequest): Promise<DiscogsApiResponse> {
    try {
      console.log(`[DiscogsApi] Making ${request.method} request to ${request.url}`);

      const response = await fetch(request.url, {
        method: request.method,
        headers: request.headers,
        // Note: For collection/wantlist endpoints, these are GET requests with no body
      });

      // Extract response headers
      const responseHeaders = this.extractHeaders(response.headers);

      // Extract rate limit information
      const rateLimitInfo = this.extractRateLimitHeaders(response.headers);
      console.log(`[DiscogsApi] Rate limit info:`, rateLimitInfo);

      // Get response body
      let body: unknown;
      const contentType = response.headers.get('content-type');

      if (contentType && contentType.includes('application/json')) {
        body = await response.json();
      } else {
        body = await response.text();
      }

      const apiResponse: DiscogsApiResponse = {
        requestId: request.requestId,
        status: response.status,
        headers: responseHeaders,
        body: body,
      };

      if (!response.ok) {
        apiResponse.error = `HTTP ${response.status}: ${response.statusText}`;
        console.error(`[DiscogsApi] Request failed:`, apiResponse.error);
      } else {
        console.log(`[DiscogsApi] Request completed successfully`, {
          requestId: request.requestId,
          status: response.status,
          rateLimitRemaining: rateLimitInfo.remaining,
        });
      }

      return apiResponse;
    } catch (error) {
      console.error(`[DiscogsApi] Request error:`, error);

      return {
        requestId: request.requestId,
        status: 0,
        headers: {},
        body: null,
        error: error instanceof Error ? error.message : 'Unknown error',
      };
    }
  }

  /**
   * Extracts all response headers as a record
   */
  private extractHeaders(headers: Headers): Record<string, string> {
    const headerRecord: Record<string, string> = {};

    headers.forEach((value, key) => {
      headerRecord[key.toLowerCase()] = value;
    });

    return headerRecord;
  }

  /**
   * Extracts Discogs rate limit headers specifically
   */
  private extractRateLimitHeaders(headers: Headers): RateLimitInfo {
    const remaining = parseInt(headers.get('x-discogs-ratelimit-remaining') || '0', 10);
    const limit = parseInt(headers.get('x-discogs-ratelimit-limit') || '60', 10);
    const window = parseInt(headers.get('x-discogs-ratelimit-window') || '60', 10);

    // Calculate reset time
    const resetTime = new Date(Date.now() + (window * 1000));

    return {
      remaining,
      limit,
      window,
      resetTime,
    };
  }

  /**
   * Validates if a request URL is a legitimate Discogs API URL
   */
  private isValidDiscogsUrl(url: string): boolean {
    try {
      const urlObj = new URL(url);
      return urlObj.hostname === 'api.discogs.com' && urlObj.protocol === 'https:';
    } catch {
      return false;
    }
  }

  /**
   * Validates a Discogs API request before making it
   */
  validateRequest(request: DiscogsApiRequest): string | null {
    if (!request.requestId) {
      return 'Request ID is required';
    }

    if (!request.url) {
      return 'URL is required';
    }

    if (!this.isValidDiscogsUrl(request.url)) {
      return 'Invalid Discogs API URL';
    }

    if (!request.method) {
      return 'HTTP method is required';
    }

    if (!['GET', 'POST', 'PUT', 'DELETE'].includes(request.method.toUpperCase())) {
      return 'Invalid HTTP method';
    }

    if (!request.headers || typeof request.headers !== 'object') {
      return 'Headers must be provided as an object';
    }

    // Check for required Discogs headers
    const authHeader = Object.keys(request.headers).find(key =>
      key.toLowerCase() === 'authorization'
    );

    if (!authHeader) {
      return 'Authorization header is required for Discogs API';
    }

    const userAgentHeader = Object.keys(request.headers).find(key =>
      key.toLowerCase() === 'user-agent'
    );

    if (!userAgentHeader) {
      return 'User-Agent header is required for Discogs API';
    }

    return null; // Valid request
  }

  /**
   * Creates a formatted error response
   */
  createErrorResponse(requestId: string, error: string): DiscogsApiResponse {
    return {
      requestId,
      status: 0,
      headers: {},
      body: null,
      error,
    };
  }
}

// Export a singleton instance
export const discogsApiService = new DiscogsApiService();