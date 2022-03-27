package controllers

import (
	"testing"
)

// Yes, but that's the standard in Go. You mix _test files and non-_test files in a package, and test some/path/packagename, not some/path/packagename/test or some/path/tests/packagename.
func TestWebsiteProcessing(t *testing.T) {
	if resp := processWebsite("http://www.linkedin.com/", "RU", 12389); !resp {
		t.Fatalf(`processWebsite("http://www.linkedin.com/", "RU", 12389) should return true, returned: %v`, resp)
	}
	// TODO: Include anomaly like kevin
	// if resp := processWebsite("https://www.linkedin.com/", "RU", 12389); !resp {
	// 	t.Fatalf(`processWebsite("https://www.linkedin.com/", "RU", 12389) should return true, returned: %v`, resp)
	// }
}
