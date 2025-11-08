# Production Dockerfile for Valkey
FROM valkey/valkey:9.0.0-alpine

# Create valkey user and group if they don't exist
RUN addgroup -g 999 valkey 2>/dev/null || true && \
    adduser -D -s /bin/sh -u 999 -G valkey valkey 2>/dev/null || true

# Create necessary directories
RUN mkdir -p /data /var/log/valkey && \
    chown -R valkey:valkey /data /var/log/valkey

# Copy production configuration
COPY valkey.prod.conf /usr/local/etc/valkey/valkey.conf

# Set proper permissions for config
RUN chown valkey:valkey /usr/local/etc/valkey/valkey.conf && \
    chmod 644 /usr/local/etc/valkey/valkey.conf

# Health check - more frequent for production monitoring
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD valkey-cli ping || exit 1

# Expose the standard Valkey port (6379 for production)
EXPOSE 6379

# Switch to valkey user for security
USER valkey

# Use custom config
CMD ["valkey-server", "/usr/local/etc/valkey/valkey.conf"]
