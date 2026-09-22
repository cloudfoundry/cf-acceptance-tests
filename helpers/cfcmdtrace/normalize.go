package cfcmdtrace

import (
	"regexp"
	"strings"
)

var (
	reGUID = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
	reHex  = regexp.MustCompile(`\b[0-9a-f]{16,}\b`)
	reSuf  = regexp.MustCompile(`-\d+$`)
)

// signature builds verb + normalized args. Rules apply most-specific first:
// prefix-name pattern, then GUID, then bare long hex, then trailing numeric suffix.
func signature(args []string, namePrefix string) string {
	rePrefix := regexp.MustCompile(
		`^` + regexp.QuoteMeta(namePrefix) + `-\d+-[A-Za-z0-9]+-[0-9a-f]{16}$`)

	verb := verbOf(args)
	out := make([]string, 0, len(args))
	out = append(out, verb)
	for _, a := range args {
		if a == verb {
			continue
		}
		out = append(out, normalizeToken(a, rePrefix))
	}
	return strings.Join(out, " ")
}

func normalizeToken(tok string, rePrefix *regexp.Regexp) string {
	if rePrefix.MatchString(tok) {
		return "<PREFIX>-N-<RESOURCE>-<RAND>"
	}
	if reGUID.MatchString(tok) {
		return reGUID.ReplaceAllString(tok, "<GUID>")
	}
	if reHex.MatchString(tok) {
		return reHex.ReplaceAllString(tok, "<HEX>")
	}
	if reSuf.MatchString(tok) {
		return reSuf.ReplaceAllString(tok, "-<N>")
	}
	return tok
}
