package story

// IncludedAssetIDs returns photographs the Story Director kept in the story.
// Omitted assets are deliberately excluded so later creative stages cannot
// silently undo an editorial omission.
func IncludedAssetIDs(spec Spec) map[string]struct{} {
	result := make(map[string]struct{})
	add := func(assetID string) {
		if assetID != "" {
			result[assetID] = struct{}{}
		}
	}
	add(spec.HeroAssetID)
	add(spec.ClimaxAssetID)
	for _, assetID := range spec.EndingAssetIDs {
		add(assetID)
	}
	for _, chapter := range spec.Chapters {
		for _, assetID := range chapter.FeaturedAssets {
			add(assetID)
		}
		for _, assetID := range chapter.SupportingAssets {
			add(assetID)
		}
	}
	return result
}
