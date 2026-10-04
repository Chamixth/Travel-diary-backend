package experience

import (
	"testing"
	"travel-diary-backend/internal/models"
)

func TestFromLegacyAlbumProducesValidExperience(t *testing.T) {
	t.Parallel()
	plan := models.AlbumPlan{Title: "After the Rain", Tone: "quiet", Chapters: []models.AlbumChapter{{
		ID: "arrival", Title: "Arrival", Blocks: []models.AlbumBlock{
			{Type: "album_cover", ImageIDs: []string{"photo-1"}, Title: "After the Rain"},
			{Type: "editorial_grid", ImageIDs: []string{"photo-2", "photo-3", "photo-4"}, Text: "Mist followed us into the hills."},
			{Type: "closing_frame", ImageIDs: []string{"photo-1"}, Title: "Going home"},
		},
	}}}
	images := []models.TripImage{
		{ID: "photo-1", FileName: "summit.jpg"}, {ID: "photo-2", FileName: "train.jpg"},
		{ID: "photo-3", FileName: "mist.jpg"}, {ID: "photo-4", FileName: "platform.jpg"},
	}
	spec := FromLegacyAlbum("experience-1", plan, images)
	known := setOf("photo-1", "photo-2", "photo-3", "photo-4")
	if err := Validate(spec, known, V1Registry()); err != nil {
		t.Fatalf("expected valid adapted experience: %v", err)
	}
	if spec.Scenes[1].Layout != "minimal-gallery" {
		t.Fatalf("expected grid to use minimal-gallery, got %q", spec.Scenes[1].Layout)
	}
	if spec.DNA.SignatureMoment.MemoryTest == "" {
		t.Fatal("expected a stated memorable moment")
	}
}
