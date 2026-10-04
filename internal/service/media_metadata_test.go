package service

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"travel-diary-backend/internal/models"
)

func TestBuildMediaAssetMetadataExtractsStableVisualSignals(t *testing.T) {
	t.Parallel()
	input := image.NewRGBA(image.Rect(0, 0, 160, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 160; x++ {
			if x < 80 {
				input.Set(x, y, color.RGBA{R: 230, G: 40, B: 30, A: 255})
			} else {
				input.Set(x, y, color.RGBA{R: 30, G: 150, B: 220, A: 255})
			}
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, input, &jpeg.Options{Quality: 92}); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}

	asset, err := BuildMediaAssetMetadata(encoded.Bytes(), "image-1", "trip-1", "user-1", "trips/trip-1/original.jpg", "image/jpeg", 160, 80)
	if err != nil {
		t.Fatalf("build metadata: %v", err)
	}
	if asset.MetadataVersion != models.MediaMetadataVersion || asset.MetadataStatus != "complete" {
		t.Fatalf("unexpected metadata state: %#v", asset)
	}
	if asset.Orientation != "landscape" || asset.DisplayWidth != 160 || asset.DisplayHeight != 80 {
		t.Fatalf("unexpected orientation: %#v", asset)
	}
	if len(asset.ContentHash) != 64 || len(asset.PerceptualHash) != 16 {
		t.Fatalf("expected stable hashes, got content=%q perceptual=%q", asset.ContentHash, asset.PerceptualHash)
	}
	if len(asset.DominantColors) == 0 || asset.Quality.Score <= 0 {
		t.Fatalf("expected visual signals, got colors=%#v quality=%#v", asset.DominantColors, asset.Quality)
	}
	if len(asset.Derivatives) != 1 || asset.Derivatives[0].Kind != "original" {
		t.Fatalf("expected original derivative, got %#v", asset.Derivatives)
	}
}

func TestDisplayOrientation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		width, height int
		want          string
	}{
		{width: 160, height: 80, want: "landscape"},
		{width: 80, height: 160, want: "portrait"},
		{width: 100, height: 100, want: "square"},
	} {
		if got := displayOrientation(test.width, test.height); got != test.want {
			t.Errorf("displayOrientation(%d,%d) = %q, want %q", test.width, test.height, got, test.want)
		}
	}
}
