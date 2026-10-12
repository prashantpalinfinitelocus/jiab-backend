package utils

import "testing"

func TestDisplayName(t *testing.T) {
	s := func(v string) *string { return &v }
	cases := []struct {
		name string
		in   *string
		want *string
	}{
		{"nil stays nil", nil, nil},
		{"empty has no name", s(""), nil},
		{"spaces only has no name", s("   "), nil},
		{"plain name kept", s("Prashant Pal"), s("Prashant Pal")},
		{"outer spaces trimmed", s("  Prashant Pal  "), s("Prashant Pal")},
		{"repeated spaces collapsed", s("Prashant   Pal"), s("Prashant Pal")},
		{"single name kept", s("Prashant"), s("Prashant")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := DisplayName(c.in)
			switch {
			case got == nil && c.want == nil:
			case got == nil || c.want == nil:
				t.Fatalf("got %v, want %v", got, c.want)
			case *got != *c.want:
				t.Fatalf("got %q, want %q", *got, *c.want)
			}
		})
	}
}
