# Go 1.25 Experimental Features

This document describes the Go 1.25 experimental features implemented in Waugzee and how to configure them.

## Overview

Go 1.25 introduces two significant experimental features that we've enabled by default:

1. **Green Tea GC** (`greenteagc`) - New garbage collector
2. **JSON v2** (`jsonv2`) - Optimized JSON encoding/decoding

## Features

### 1. Green Tea GC (greenteagc)

**Performance Impact**: 10-40% reduction in garbage collection overhead

**What it does**:
- Improves the performance of marking and scanning small objects
- Better locality and CPU scalability
- Particularly beneficial for applications that heavily use the garbage collector

**Status**: Experimental - design expected to evolve in future releases

### 2. JSON v2 (jsonv2)

**Performance Impact**: Substantially faster JSON decoding, encoding at parity

**What it does**:
- Major revision of the standard `encoding/json` package
- New packages available:
  - `encoding/json/v2` - major revision of the standard JSON package
  - `encoding/json/jsontext` - lower-level JSON syntax processing
- When enabled, the standard `encoding/json` package uses the new implementation
- Marshaling/unmarshaling behavior remains unchanged (drop-in replacement)

**Status**: Experimental - encourages testing and feedback

## Configuration

### Default Configuration

Both features are **enabled by default** in development and production:

```bash
GOEXPERIMENT=greenteagc,jsonv2
```

### Customizing Features

You can customize which experimental features are enabled using the `GOEXPERIMENT` environment variable.

#### Option 1: Using .env file

Add to your `.env` file:

```bash
# Enable both features (default)
GOEXPERIMENT=greenteagc,jsonv2

# Enable only Green Tea GC
GOEXPERIMENT=greenteagc

# Enable only JSON v2
GOEXPERIMENT=jsonv2

# Disable all experimental features
GOEXPERIMENT=none
```

#### Option 2: Using .env.local file

For local development overrides, add to `.env.local` (see `.env.local.example` for full documentation):

```bash
GOEXPERIMENT=greenteagc,jsonv2
```

#### Option 3: Build-time Override

Override at build time:

```bash
docker compose -f docker-compose.dev.yml build --build-arg GOEXPERIMENT=greenteagc server
```

#### Option 4: Production Deployment

For production builds (Dockerfile), the same `GOEXPERIMENT` environment variable or build arg can be used:

```bash
docker build --build-arg GOEXPERIMENT=greenteagc,jsonv2 -f server/Dockerfile .
```

## Implementation Details

### Files Modified

1. **server/Dockerfile** - Production build configuration
2. **server/Dockerfile.dev** - Development build configuration
3. **docker-compose.dev.yml** - Development orchestration
4. **.env.local.example** - Configuration documentation
5. **Tiltfile** - Development environment configuration

### How It Works

1. **Build Time**: The `GOEXPERIMENT` build argument is passed to the Go compiler during the build phase
2. **Runtime**: The `GOEXPERIMENT` environment variable is set in the container for Air hot-reloading
3. **Default Value**: If not specified, defaults to `greenteagc,jsonv2`

### Verification

You can verify which experiments are active by checking the Tilt dashboard output:

```
⚡ Go Experiments: greenteagc,jsonv2
```

Or check the environment inside the container:

```bash
docker compose -f docker-compose.dev.yml exec server printenv GOEXPERIMENT
```

## Testing and Feedback

Since these features are experimental:

1. **Monitor Performance**: Watch for any unexpected behavior or performance changes
2. **Report Issues**: If you encounter issues, report them to the Go team via the GitHub issue tracker
3. **Easy Rollback**: Simply set `GOEXPERIMENT=none` to disable all experimental features
4. **Gradual Adoption**: You can enable features individually to isolate any issues

## Performance Monitoring

### Metrics to Watch

With Green Tea GC:
- GC pause times
- CPU usage during garbage collection
- Memory usage patterns

With JSON v2:
- JSON encoding/decoding latency
- CPU usage during JSON operations
- Overall API response times

### Recommended Approach

1. **Development**: Keep both features enabled (default) to test and provide feedback
2. **Staging**: Monitor performance metrics compared to baseline
3. **Production**: Enable after thorough testing in staging environment

## Disabling Features

If you need to disable the experimental features:

```bash
# In .env or .env.local
GOEXPERIMENT=none
```

Then rebuild your containers:

```bash
tilt down
tilt up
```

## References

- [Go 1.25 Release Notes](https://go.dev/doc/go1.25)
- Go GitHub Issue Tracker: https://github.com/golang/go/issues

## Notes

- These are **experimental features** - the design may evolve in future releases
- The Go team actively encourages testing and feedback
- Both features are considered stable enough for non-critical applications
- Performance gains are expected in real-world programs that heavily use GC and JSON operations
