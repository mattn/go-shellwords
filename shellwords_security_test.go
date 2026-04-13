package shellwords

import (
	"strings"
	"testing"
)

// ParseBacktick=true → an unmatched ‘)’ should be treated as an ERROR (not a literal)
func TestUnmatchedClosingParen_ReturnsError_ParseBacktickEnabled(t *testing.T) {
	p := NewParser()
	p.ParseBacktick = true

	_, err := p.Parse("))")
	if err == nil {
		t.Fatal("expected error for unmatched ')', got nil")
	}
}

// ParseBacktick=true → Do not execute; must fail
func TestBareClosingParen_NoExecution_ReturnsError(t *testing.T) {
	p := NewParser()
	p.ParseBacktick = true

	_, err := p.Parse("prefix)echo PWNED)")
	if err == nil {
		t.Fatal("expected error for unmatched ')', got nil")
	}
}

// Default behavior → bare ‘)’ is a syntax error, consistent with ‘(‘ handling
func TestBareClosingParen_DefaultParsingError(t *testing.T) {
	_, err := Parse(")a b)c d")
	if err == nil {
		t.Fatal("expected error for unmatched ‘)’, got nil")
	}
}

// ParseBacktick=false → $(...) remains a FULL literal
func TestDollarParenLiteralWhenParseBacktickDisabled(t *testing.T) {
	p := NewParser()
	p.ParseBacktick = false

	out, err := p.Parse(`$(echo "foo")`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{`$(echo "foo")`}
	if len(out) != len(want) {
		t.Fatalf("unexpected token count: got %d, want %d, out=%#v", len(out), len(want), out)
	}
	if out[0] != want[0] {
		t.Fatalf("unexpected output: got %q, want %q", out[0], want[0])
	}
}

// If true → $(...) continues to work with ParseBacktick=true
func TestValidDollarCommandSubstitutionStillWorks(t *testing.T) {
	p := NewParser()
	p.ParseBacktick = true

	out, err := p.Parse("prefix $(printf hello) suffix")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.Join(out, " ")
	if !strings.Contains(got, "hello") {
		t.Fatalf("expected valid command substitution to work, got: %q", got)
	}
}

// Error → Unclosed $(
func TestUnclosedDollarCommandSubstitutionReturnsError(t *testing.T) {
	p := NewParser()
	p.ParseBacktick = true

	_, err := p.Parse("prefix $(echo hello")
	if err == nil {
		t.Fatal("expected error for unclosed command substitution, got nil")
	}
}