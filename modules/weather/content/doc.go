// Package content parses the weather module's own data files (climate
// profiles, ambient emote tables) from a file system — typically the module's
// embedded files/ tree. It is pure: no engine imports (enforced by
// arch_test.go), so everything here is unit-testable standalone. The only
// non-stdlib dependency is gopkg.in/yaml.v2, which the GoMud engine itself
// depends on.
package content
