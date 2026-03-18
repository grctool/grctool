---
title: "SD-001 Appendix: AccountableHQ API Discovery"
phase: "02-design"
category: "api-discovery"
status: "Pending Validation"
tags: ["accountablehq", "api", "discovery"]
created: 2026-03-18
updated: 2026-03-18
---

# AccountableHQ API Discovery

## Status: PENDING VALIDATION

The endpoints below are **assumed** based on standard GRC platform REST API patterns.
They must be validated against the actual AccountableHQ API documentation before
the real HTTP client is implemented.

## Assumed API Endpoints

| Method | Path | Purpose | Confirmed |
|--------|------|---------|-----------|
| GET | /api/v1/policies | List policies (paginated) | [ ] |
| GET | /api/v1/policies/{id} | Get single policy | [ ] |
| POST | /api/v1/policies | Create policy | [ ] |
| PUT | /api/v1/policies/{id} | Update policy | [ ] |
| DELETE | /api/v1/policies/{id} | Delete policy | [ ] |

## Assumed Authentication

| Method | Details | Confirmed |
|--------|---------|-----------|
| API Key | `X-API-Key` header or `Authorization: Bearer <token>` | [ ] |
| OAuth2 | Client credentials flow | [ ] |

## Assumed Response Format

```json
{
  "data": [
    {
      "id": "pol-abc-123",
      "title": "Access Control Policy",
      "content": "# Policy Content\n\n...",
      "status": "active",
      "version": 3,
      "category": "Security",
      "owner": "compliance@example.com",
      "review_date": "2026-06-01",
      "created_at": "2025-01-15T10:00:00Z",
      "updated_at": "2026-03-01T14:30:00Z"
    }
  ],
  "meta": {
    "page": 1,
    "per_page": 25,
    "total": 42
  }
}
```

## Validation Checklist

- [ ] Obtain API documentation from AccountableHQ
- [ ] Confirm endpoint paths and HTTP methods
- [ ] Confirm authentication mechanism
- [ ] Confirm response envelope format (data/meta vs flat array)
- [ ] Confirm pagination parameters (page/per_page vs offset/limit)
- [ ] Confirm policy field names and types
- [ ] Confirm rate limiting headers and quotas
- [ ] Record VCR cassettes from real API responses
- [ ] Update SD-001 with confirmed endpoints
- [ ] Update AHQPolicy struct in provider.go if field names differ

## Implementation Note

The `AccountableHQClient` interface in `internal/providers/accountablehq/provider.go`
is designed to be API-agnostic. The real HTTP client (implementing this interface)
will be created once the API is confirmed. Until then, all tests use the
`stubAHQClient` which validates the provider logic independent of the API shape.
