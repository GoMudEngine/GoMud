// Package engine adapts the GoMud engine to the weather module's pure core. It
// concentrates the direct engine-world calls — room/zone/biome reads via
// internal/rooms and the on-disk graph-cache codec — and implements
// crawler.WorldReader. Together with the root weather package (which imports
// internal/* for plugin infrastructure: plugins, events, users, mudlog), these
// are the only packages that touch the engine; sim/ and crawler/ stay pure.
// That split is what keeps the module portable across GoMud and DOGMud.
package engine
