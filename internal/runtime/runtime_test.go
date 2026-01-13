package runtime

import (
	"testing"
)

func TestMeetsRequirement(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		requirement string
		want        bool
	}{
		// Empty requirement
		{"empty requirement", "18.0.0", "", true},

		// Greater than or equal
		{"node 18 meets >=18", "18.0.0", ">=18", true},
		{"node 20 meets >=18", "20.0.0", ">=18", true},
		{"node 17 fails >=18", "17.0.0", ">=18", false},
		{"node 18.5 meets >=18.5", "18.5.0", ">=18.5", true},
		{"node 18.4 fails >=18.5", "18.4.0", ">=18.5", false},

		// Greater than
		{"node 19 meets >18", "19.0.0", ">18", true},
		{"node 18 fails >18", "18.0.0", ">18", false},

		// Less than or equal
		{"node 18 meets <=18", "18.0.0", "<=18", true},
		{"node 17 meets <=18", "17.0.0", "<=18", true},
		{"node 19 fails <=18", "19.0.0", "<=18", false},

		// Less than
		{"node 17 meets <18", "17.0.0", "<18", true},
		{"node 18 fails <18", "18.0.0", "<18", false},

		// Exact match
		{"exact match succeeds", "18.0.0", "=18.0.0", true},
		{"exact match fails", "18.0.1", "=18.0.0", false},

		// Python versions
		{"python 3.10 meets >=3.10", "3.10.0", ">=3.10", true},
		{"python 3.9 fails >=3.10", "3.9.0", ">=3.10", false},
		{"python 3.11 meets >=3.10", "3.11.5", ">=3.10", true},

		// Implicit >= operator
		{"implicit >= succeeds", "18.0.0", "18", true},
		{"implicit >= fails", "17.0.0", "18", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MeetsRequirement(tt.version, tt.requirement); got != tt.want {
				t.Errorf("MeetsRequirement(%q, %q) = %v, want %v", tt.version, tt.requirement, got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		version string
		want    []int
	}{
		{"18", []int{18}},
		{"18.0", []int{18, 0}},
		{"18.0.0", []int{18, 0, 0}},
		{"3.10.5", []int{3, 10, 5}},
		{"20.11.0", []int{20, 11, 0}},
		{"1.0.0-beta", []int{1, 0, 0}}, // strips suffix
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := parseVersion(tt.version)
			if len(got) != len(tt.want) {
				t.Errorf("parseVersion(%q) = %v, want %v", tt.version, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseVersion(%q)[%d] = %v, want %v", tt.version, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCompareVersionParts(t *testing.T) {
	tests := []struct {
		a    []int
		b    []int
		want int
	}{
		{[]int{18}, []int{18}, 0},
		{[]int{18, 0, 0}, []int{18}, 0},
		{[]int{18}, []int{18, 0, 0}, 0},
		{[]int{19}, []int{18}, 1},
		{[]int{18}, []int{19}, -1},
		{[]int{18, 1}, []int{18, 0}, 1},
		{[]int{18, 0}, []int{18, 1}, -1},
		{[]int{3, 10}, []int{3, 9}, 1},
		{[]int{3, 9}, []int{3, 10}, -1},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := compareVersionParts(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareVersionParts(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
