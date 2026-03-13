## Sukumad Rate-Limiting Correction

### What changed

Outbound Sukumad traffic now uses a single destination-scoped limiter owned by [backend/internal/ratelimit](/Users/sam/projects/go/sukumadpro/backend/internal/ratelimit/ratelimit.go).

The active outbound path is:

1. request creation or retry selects a delivery attempt
2. [backend/internal/sukumad/delivery/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/delivery/service.go) orchestrates the submission lifecycle
3. [backend/internal/sukumad/dhis2/service.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2/service.go) resolves the destination key
4. [backend/internal/sukumad/dhis2/client.go](/Users/sam/projects/go/sukumadpro/backend/internal/sukumad/dhis2/client.go) waits on the shared limiter immediately before `http.Client.Do(...)`
5. response handling, async-task creation, retry state, and observability continue through the existing delivery and async services

This means Sukumad no longer issues outbound HTTP requests from the DHIS2 client without first passing through the shared limiter.

### Covered paths

The limiter now applies to:

- initial request-triggered submissions
- retry submissions created from failed deliveries
- manual resubmissions that reuse the delivery submission service
- async follow-up traffic to destination systems through the DHIS2 poller

The dedicated worker process now owns send/retry/poll execution, and any path that reaches the shared DHIS2 outbound client is rate limited by the same registry.

### Configuration

Backend config now supports destination-specific outbound limits under [backend/internal/config/config.go](/Users/sam/projects/go/sukumadpro/backend/internal/config/config.go) and [backend/config/config.yaml](/Users/sam/projects/go/sukumadpro/backend/config/config.yaml):

```yaml
sukumad:
  rate_limit:
    default:
      requests_per_second: 2
      burst: 2
    destinations:
      dhis2-ug:
        requests_per_second: 5
        burst: 10
```

Rules:

- destination keys should match the persisted Sukumad server code
- when no destination override exists, the default policy is used
- async polling uses the same destination code carried through the async repository projection; host fallback is only used if no destination code is available
- config is read from the active Viper-backed runtime snapshot, so updated values are picked up without rebuilding limiter call sites

### Operational notes

- limiter waits respect request context cancellation
- a cancelled or expired wait returns through the normal delivery error path; it is not silently dropped
- existing request, delivery, async-task, and worker observability remains intact because the delivery/async services still own status transitions and event recording
- secrets remain masked by the existing observability sanitization paths
