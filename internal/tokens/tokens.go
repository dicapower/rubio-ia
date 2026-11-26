// Package tokens gives a rough token count. It is deliberately simple: the
// point is to catch instructions that grow, not to bill anyone.
package tokens

// Estimate counts about four bytes per token, rounded up. Real tokenizers
// differ by tens of percent, which is fine for a budget guard.
func Estimate(s string) int {
	return (len(s) + 3) / 4
}
