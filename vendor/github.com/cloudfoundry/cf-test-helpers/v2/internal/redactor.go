package internal

import "strings"

type Redactor interface {
	Redact(toRedact string) string
}

type redactor struct {
	redactees []string
}

func NewRedactor(redactees ...string) Redactor {
	return &redactor{
		redactees: redactees,
	}
}

func (r *redactor) Redact(toRedact string) string {
	for _, r := range r.redactees {
		toRedact = strings.ReplaceAll(toRedact, r, "[REDACTED]")
	}

	return toRedact
}
