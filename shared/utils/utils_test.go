package utils

import (
	"fmt"
	"testing"
)

func TestGetGeoIPData(t *testing.T) {
	c, iso, err := GetGeoIPData("1.1.1.1")
	if err != nil {
		t.Errorf("GeoIPLookup failed: %s", err.Error())
	}
	fmt.Printf("%s , %s\n", c, iso)
}