// filepath: internal/media/image_preview_test.go
// (Moved from internal/storage/preview_test.go)
package media

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// createTestEntry creates an in-memory entry buffer
func createTestEntry(t *testing.T, width, height int) *bytes.Buffer {
	t.Helper()
	// Add check to prevent helper from failing on invalid test setup
	if width <= 0 || height <= 0 {
		t.Fatalf("createTestEntry helper: invalid dimensions %dx%d", width, height)
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill entry with a color to make it non-zero
	blue := color.RGBA{0, 0, 255, 255}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, blue)
		}
	}

	buf := new(bytes.Buffer)
	err := png.Encode(buf, img)
	if err != nil {
		t.Fatalf("Failed to encode test entry: %v", err)
	}
	return buf
}

func TestCreateImagePreview(t *testing.T) {
	testCases := []struct {
		name           string
		origWidth      int
		origHeight     int
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "Landscape Entry (600x400)",
			origWidth:      600,
			origHeight:     400,
			expectedWidth:  200,
			expectedHeight: 133,
		},
		{
			name:           "Portrait Entry (400x600)",
			origWidth:      400,
			origHeight:     600,
			expectedWidth:  133,
			expectedHeight: 200,
		},
		{
			name:           "Square Entry (500x500)",
			origWidth:      500,
			origHeight:     500,
			expectedWidth:  200,
			expectedHeight: 200,
		},
		{
			name:           "Small Entry (100x50) - Does not scale up",
			origWidth:      100,
			origHeight:     50,
			expectedWidth:  100, // Stays 100
			expectedHeight: 50,  // Stays 50
		},
		{
			name:           "Small Portrait (50x100) - Does not scale up",
			origWidth:      50,
			origHeight:     100,
			expectedWidth:  50,
			expectedHeight: 100,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Create in-memory entry
			entryBuf := createTestEntry(t, tc.origWidth, tc.origHeight)

			// 2. Define temp output file
			const previewPath = "test_preview_output.jpg"
			defer os.Remove(previewPath) // Clean up

			// 3. Call the function (renamed to CreateImagePreview)
			err := CreateImagePreview(entryBuf, previewPath)
			assert.NoError(t, err)

			// 4. Open and decode the result
			f, err := os.Open(previewPath)
			assert.NoError(t, err)
			defer f.Close()

			previewImg, format, err := image.Decode(f)
			assert.NoError(t, err)

			// 5. Check format and dimensions
			assert.Equal(t, "jpeg", format, "Preview was not saved as a JPEG")
			bounds := previewImg.Bounds()
			assert.Equal(t, tc.expectedWidth, bounds.Dx(), "Preview width is incorrect")
			assert.Equal(t, tc.expectedHeight, bounds.Dy(), "Preview height is incorrect")
		})
	}
}

// TestCreateImagePreview_InvalidData tests for robustness against invalid entry data
func TestCreateImagePreview_InvalidData(t *testing.T) {
	// Pass invalid data directly instead of trying to create a 0x0 entry
	entryBuf := bytes.NewBuffer([]byte("this is not a valid entry"))
	const previewPath = "test_preview_invalid.jpg"
	defer os.Remove(previewPath)

	// Call the function we are testing
	err := CreateImagePreview(entryBuf, previewPath)

	// Assert that it failed gracefully
	assert.Error(t, err, "Should have failed for invalid entry data")
	if err != nil {
		assert.Contains(t, err.Error(), "could not decode entry for preview", "Error message mismatch")
	}

	// Check that the file was not created
	_, err = os.Stat(previewPath)
	assert.True(t, os.IsNotExist(err), "Preview file should not exist after a failure")
}
