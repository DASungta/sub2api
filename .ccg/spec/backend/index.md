# Backend Specifications

## Authentication Cache Projections

When a `Group` field affects request-time routing, admission, or billing, add it to `APIKeyAuthGroupSnapshot`, map it in both snapshot directions, bump `apiKeyAuthSnapshotVersion`, and add a JSON round-trip regression test. Repository projection coverage alone is insufficient because authenticated requests use the materialized cache snapshot.

## Default-Enabled API Fields

Represent default-enabled create-request booleans as `*bool` through the handler and service input. Treat `nil` as the documented default and preserve explicit `false`; a plain `bool` cannot distinguish omission from an explicit disable.
