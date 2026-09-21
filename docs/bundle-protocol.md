# Bundle protocol

The registry publishes one manifest and zero or more immutable, version-addressed pages. All
limits are configured by the decision service; registry values are untrusted input.

## Manifest

```json
{
  "revision": 2,
  "page_count": 2,
  "rule_count": 4,
  "digest": "sha256:0123456789abcdef..."
}
```

`GET /v1/policy-bundles/current` returns the manifest and an HTTP `ETag`. It accepts
`If-None-Match` and may return `304 Not Modified`.

## Page

```json
{
  "revision": 2,
  "page_index": 0,
  "rules": [
    {
      "effect": "deny",
      "id": "deny-restricted-region",
      "operation": "*",
      "priority": 10,
      "region": "restricted"
    }
  ]
}
```

Pages are fetched at:

```text
GET /v1/policy-bundles/{revision}/pages/{page_index}
```

Page indexes are zero-based. Every returned page must agree with both the requested revision and
index. Rule IDs are unique across the complete bundle.

## Canonical digest

The digest covers the logical rule set rather than JSON whitespace or page boundaries:

1. Sort rules by ascending `priority`, then ascending `id`.
2. Serialize the array as compact UTF-8 JSON.
3. Object keys are ordered `effect`, `id`, `operation`, `priority`, `region`.
4. Hash the bytes with SHA-256 and prefix the lower-case hexadecimal value with `sha256:`.

The Go codebase includes a canonical digest implementation used by its policy domain model.

## Structural validity

A candidate is structurally invalid when any of these conditions holds:

- Revision, page count, or rule count is negative.
- Page count exceeds the configured maximum.
- A page has a different revision or index than requested.
- The assembled rule count differs from the manifest.
- A rule ID is empty or duplicated.
- An effect is not `allow` or `deny`.
- An operation or region is empty.
- Total decoded response bytes exceed the configured maximum.
- The canonical digest does not equal the manifest digest.

The registry client already rejects malformed JSON, unknown fields, non-success statuses, and
individual responses larger than its configured read limit.
