package shellwords

import (
	"strings"
	"testing"
)

func TestUnmatchedClosingParen_NoPanic_ParseBacktickEnabled(t *testing.T) {
	p := NewParser()
	p.ParseBacktick = true

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	out, err := p.Parse("))")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(out) != 1 || out[0] != "))" {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestBareClosingParen_DoesNotTriggerCommandExecution(t *testing.T) {
	p := NewParser()
	p.ParseBacktick = true

	out, err := p.Parse("prefix)echo PWNED)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.Join(out, " ")
	if strings.Contains(got, "PWNED") && got != "prefix)echo PWNED)" {
		t.Fatalf("unexpected command execution or substitution occurred: %q", got)
	}

	want := []string{"prefix)echo", "PWNED)"}
	if len(out) != len(want) {
		t.Fatalf("unexpected token count: got %d, want %d, out=%#v", len(out), len(want), out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("unexpected token at %d: got %q, want %q, out=%#v", i, out[i], want[i], out)
		}
	}
}

func TestBareClosingParen_DefaultParsingLiteral(t *testing.T) {
	out, err := Parse(")a b)c d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{")a", "b)c", "d"}
	if len(out) != len(want) {
		t.Fatalf("unexpected token count: got %d, want %d, out=%#v", len(out), len(want), out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("unexpected token at %d: got %q, want %q, out=%#v", i, out[i], want[i], out)
		}
	}
}

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

func TestUnclosedDollarCommandSubstitutionReturnsError(t *testing.T) {
	p := NewParser()
	p.ParseBacktick = true

	_, err := p.Parse("prefix $(echo hello")
	if err == nil {
		t.Fatal("expected error for unclosed command substitution, got nil")
	}
}
