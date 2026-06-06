package lowfat

// EstimateTokens approximates token count as ceil(len/4) — lowfat's metric
// (matches its bash `(len + 3) / 4`). Used for savings reporting and the
// small-output skip threshold.
func EstimateTokens(s string) int {
	return (len(s) + 3) / 4
}
