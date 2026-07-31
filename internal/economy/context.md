# Economy Context

## Purpose

`internal/economy` is a tiny classification package: it maps an item id to an
**economic bucket**, the named category the logistics layer routes goods by.

It is deliberately minimal and dependency-light so that shops, caravans,
ferries, foragers, and warehouses can all agree on "what kind of goods is this"
without importing each other.

The health/observability side of the economy lives in the sub-package
[`economy/health`](health/context.md).

## Files

- **buckets.go** — the whole package.

## Public API

```go
func BucketFor(itemId int) string
func AllBuckets() []string
```

## How buckets are used

Caravan and ferry route definitions declare delivery and pickup buckets by
name. A runner visiting a vendor delivers everything in its outbound buckets
and collects anything matching its pickup buckets — so adding a new item to an
existing bucket automatically puts it into circulation on every route that
already carries that bucket, with no route edits.

That leverage cuts both ways: **re-bucketing an existing item silently changes
which caravans carry it.** Check the route definitions before moving one.

## Gotchas

- **`BucketFor` is total** — every item id gets a bucket, including ids that do
  not exist. Do not use it as an existence check.
- **`AllBuckets()` is the authoritative list.** Route YAML naming a bucket not
  in this list will match nothing, and nothing warns — validate against
  `AllBuckets()` when adding a route.

## Dependencies

`items`.

## Consumers

`caravan`, `ferry`, `shops`, `warehouse`, `forager`, and `economy/health`.
