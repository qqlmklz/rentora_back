package models

import "testing"

func TestNormalizePropertySubcategoryAPI_Studio(t *testing.T) {
	if got := NormalizePropertySubcategoryAPI("studio"); got != PropertyTypeStoredStudio {
		t.Fatalf("studio: got %q want %q", got, PropertyTypeStoredStudio)
	}
	if got := NormalizePropertySubcategoryAPI("STUDIO"); got != PropertyTypeStoredStudio {
		t.Fatalf("STUDIO: got %q", got)
	}
}

func TestIsPropertyTypeAllowedForCategory_StudioResidential(t *testing.T) {
	if !IsPropertyTypeAllowedForCategory("residential", "studio") {
		t.Fatal("residential + studio should be allowed")
	}
	if !IsPropertyTypeAllowedForCategory(CategoryStoredResidential, PropertyTypeStoredStudio) {
		t.Fatal("жилая + студия should be allowed")
	}
}

func TestIsPropertyTypeAllowedForCategory_StudioNotCommercial(t *testing.T) {
	if IsPropertyTypeAllowedForCategory("commercial", "studio") {
		t.Fatal("commercial + studio should be rejected")
	}
}

func TestNormalizePropertyTypeCatalogFilter_Studio(t *testing.T) {
	if got := NormalizePropertyTypeCatalogFilter("studio"); got != PropertyTypeStoredStudio {
		t.Fatalf("filter studio: got %q want %q", got, PropertyTypeStoredStudio)
	}
}

func TestPropertyTypeRequiresRoomCount(t *testing.T) {
	if PropertyTypeRequiresRoomCount("residential", "studio") {
		t.Fatal("studio should not require rooms count")
	}
	if !PropertyTypeRequiresRoomCount("residential", "apartment") {
		t.Fatal("apartment should require rooms count")
	}
	if PropertyTypeRequiresRoomCount("commercial", "office") {
		t.Fatal("commercial should not require rooms count")
	}
}
