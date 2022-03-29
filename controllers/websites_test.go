package controllers

import (
	"strings"
	"testing"
)

// Yes, but that's the standard in Go. You mix _test files and non-_test files in a package, and test some/path/packagename, not some/path/packagename/test or some/path/tests/packagename.
func TestWebsiteProcessing(t *testing.T) {
	// if resp := processWebsite("http://www.linkedin.com/", "RU", 12389, 1); !resp {
	// 	t.Fatalf(`processWebsite("http://www.linkedin.com/", "RU", 12389) should return true, returned: %v`, resp)
	// }
	// if resp := processWebsite("https://www.facebook.com/", "SB", 132462, 1); resp {
	// 	t.Fatalf(`processWebsite("https://www.facebook.com/", "SB", 132462) should return false, returned: %v`, resp)
	// }
	// TODO: Include anomaly like kevin
	// if resp := processWebsite("https://www.linkedin.com/", "RU", 12389); !resp {
	// 	t.Fatalf(`processWebsite("https://www.linkedin.com/", "RU", 12389) should return true, returned: %v`, resp)
	// }
}

func TestReadWebsites(t *testing.T) {
	topWebsitesBlob := readTopWebsitesCSV("../static/topwebsites.csv")
	if topWebsitesBlob == "" {
		t.Fatalf(`readTopWebsitesCSV() == "", there should be some data in the csv returned.`)
	}
	topWebsitesArr := strings.Split(topWebsitesBlob, "\n")
	if len(topWebsitesArr) != 50 {
		t.Fatalf(`strings.Split(readTopWebsitesCSV(), "\n") != 50, there should be 50 top websites.`)
	}
}
