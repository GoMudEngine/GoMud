// Package crawler builds a zone-adjacency Graph from a read-only view of the
// world (the WorldReader interface). The traversal logic here is pure and
// engine-independent; the live WorldReader implementation that wraps
// internal/rooms lives in a separate package built only inside a GoMud checkout.
package crawler
