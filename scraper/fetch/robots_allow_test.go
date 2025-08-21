package fetch

import "testing"

func TestAllowOverridesDisallow_BoundaryAware(t *testing.T) {
	// Broad disallow with a specific allowed subpath
	rob := makeRobots("User-agent: *\nDisallow: /jobs\nAllow: /jobs/api\n")

	tests := []struct {
		path string
		want bool
	}{
		{"/", true},
		{"/jobs", false},            // blocked by Disallow: /jobs
		{"/jobs/", false},           // blocked by Disallow: /jobs
		{"/jobs/foo", false},        // blocked by Disallow: /jobs
		{"/jobs/api", true},         // explicitly allowed
		{"/jobs/api/", true},        // allowed subpath
		{"/jobs/api/v1", true},      // allowed subpath
		{"/jobs/apis", false},       // NOT a subpath of /jobs/api; remains blocked by /jobs
	}

	for _, tc := range tests {
		got := rob.isAllowed("*", tc.path)
		if got != tc.want {
			t.Errorf("isAllowed(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}
