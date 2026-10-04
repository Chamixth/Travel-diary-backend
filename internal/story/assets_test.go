package story

import "testing"

func TestIncludedAssetIDsExcludesEditorialOmissions(t *testing.T) {
	t.Parallel()
	assets := IncludedAssetIDs(Spec{
		HeroAssetID: "hero", ClimaxAssetID: "climax", EndingAssetIDs: []string{"ending"},
		Chapters:        []Chapter{{FeaturedAssets: []string{"featured"}, SupportingAssets: []string{"supporting"}}},
		OmittedAssetIDs: []string{"omitted"},
	})
	for _, expected := range []string{"hero", "climax", "ending", "featured", "supporting"} {
		if _, ok := assets[expected]; !ok {
			t.Fatalf("expected %q to be included", expected)
		}
	}
	if _, ok := assets["omitted"]; ok {
		t.Fatal("omitted asset must not be reintroduced")
	}
}
