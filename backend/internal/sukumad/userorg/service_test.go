package userorg

import "testing"

func TestScopeContainsPathUsesSegmentBoundaries(t *testing.T) {
	scope := Scope{
		Restricted:   true,
		PathPrefixes: []string{"/root/dist"},
	}

	cases := []struct {
		name string
		path string
		want bool
	}{
		{name: "exact match", path: "/root/dist", want: true},
		{name: "descendant match", path: "/root/dist/facility", want: true},
		{name: "prefix collision", path: "/root/district", want: false},
		{name: "different branch", path: "/root/other", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScopeContainsPath(scope, tc.path)
			if got != tc.want {
				t.Fatalf("ScopeContainsPath(%q) = %t, want %t", tc.path, got, tc.want)
			}
		})
	}
}
