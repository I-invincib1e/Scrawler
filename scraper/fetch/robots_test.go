package fetch

import "testing"

func makeRobots(rules string) *robotsTxt {
	rob := &robotsTxt{uaRules: map[string][]robotRule{"*": {}}}
	parseRobots(rules, rob)
	return rob
}

func TestBoundaryMatching_DisallowJobs(t *testing.T) {
	rob := makeRobots("User-agent: *\nDisallow: /jobs\n")

	tests := []struct {
		path string
		want bool
	}{
		{"/", true},          // allowed
		{"/doc/", true},      // allowed
		{"/about/", true},    // allowed
		{"/jobs", false},     // exact blocked
		{"/jobs/", false},    // subpath blocked
		{"/jobs/foo", false}, // subpath blocked
		{"/job", true},       // not blocked (prefix boundary)
		{"/jobs2", true},     // not blocked (prefix boundary)
		{"/documentation", true},
	}

	for _, tc := range tests {
		got := rob.isAllowed("*", tc.path)
		if got != tc.want {
			t.Errorf("isAllowed(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
