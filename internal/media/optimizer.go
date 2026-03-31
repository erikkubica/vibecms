package media

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/draw"

	// Register WebP decoder so image.Decode can read WebP files (pure Go, no CGO).
	_ "golang.org/x/image/webp"
)

// Normalize normalizes an uploaded image: downscale if exceeds maxDim (fit inside),
// strip metadata (by re-encoding), and compress. Returns bytes, mime type, width, height.
// Keeps original format.
func Normalize(src io.Reader, mimeType string, maxDim int, jpegQuality int) ([]byte, string, int, int, error) {
	img, err := decodeImage(src, mimeType)
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("decode: %w", err)
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Downscale if either dimension exceeds maxDim.
	if w > maxDim || h > maxDim {
		img = resizeFit(img, maxDim, maxDim)
		bounds = img.Bounds()
		w, h = bounds.Dx(), bounds.Dy()
	}

	quality := jpegQuality
	if quality <= 0 {
		quality = 92
	}

	outMime := mimeType
	if outMime == "image/webp" {
		outMime = "image/jpeg" // WebP encoding unavailable; fall back to JPEG.
	}

	data, err := encodeImage(img, outMime, quality)
	if err != nil {
		return nil, "", 0, 0, fmt.Errorf("encode: %w", err)
	}

	return data, outMime, w, h, nil
}

// Resize resizes an image to given dimensions with the specified mode.
// Mode can be "crop", "fit", or "width". Returns bytes and output mime type.
func Resize(src io.Reader, mimeType string, width, height int, mode string, quality int) ([]byte, string, error) {
	img, err := decodeImage(src, mimeType)
	if err != nil {
		return nil, "", fmt.Errorf("decode: %w", err)
	}

	if quality <= 0 {
		quality = 80
	}

	var resized image.Image
	switch mode {
	case "crop":
		resized = resizeCrop(img, width, height)
	case "fit":
		resized = resizeFit(img, width, height)
	case "width":
		resized = resizeWidth(img, width)
	default:
		return nil, "", fmt.Errorf("unknown resize mode: %s", mode)
	}

	outMime := mimeType
	if outMime == "image/webp" {
		outMime = "image/jpeg" // WebP encoding unavailable; fall back to JPEG.
	}

	data, err := encodeImage(resized, outMime, quality)
	if err != nil {
		return nil, "", fmt.Errorf("encode: %w", err)
	}

	return data, outMime, nil
}

// ConvertToWebP is a no-op stub. WebP encoding requires CGO (libwebp).
// The decoder is registered via golang.org/x/image/webp so WebP uploads can be read,
// but encoding to WebP is not available in pure-Go builds.
// WebP input images are re-encoded as JPEG instead.
func ConvertToWebP(src io.Reader, quality int) ([]byte, error) {
	return nil, fmt.Errorf("WebP encoding is not available (requires CGO); use JPEG or PNG output instead")
}

// decodeImage decodes an image from the reader based on MIME type.
func decodeImage(r io.Reader, mimeType string) (image.Image, error) {
	switch mimeType {
	case "image/jpeg":
		return jpeg.Decode(r)
	case "image/png":
		return png.Decode(r)
	case "image/gif":
		return gif.Decode(r)
	case "image/webp":
		// Handled by registered golang.org/x/image/webp decoder via image.Decode.
		img, _, err := image.Decode(r)
		return img, err
	default:
		// Fall back to Go's auto-detection.
		img, _, err := image.Decode(r)
		return img, err
	}
}

// resizeFit resizes an image to fit inside the given bounds while preserving aspect ratio.
func resizeFit(img image.Image, maxW, maxH int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= maxW && srcH <= maxH {
		return img
	}

	ratio := float64(srcW) / float64(srcH)
	newW := maxW
	newH := int(float64(newW) / ratio)

	if newH > maxH {
		newH = maxH
		newW = int(float64(newH) * ratio)
	}

	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

// resizeCrop resizes an image to cover the target dimensions, then center-crops to exact size.
func resizeCrop(img image.Image, w, h int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	// Calculate scale to cover target dimensions.
	scaleW := float64(w) / float64(srcW)
	scaleH := float64(h) / float64(srcH)
	scale := scaleW
	if scaleH > scaleW {
		scale = scaleH
	}

	// Scale up to cover.
	scaledW := int(float64(srcW) * scale)
	scaledH := int(float64(srcH) * scale)
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}

	scaled := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), img, bounds, draw.Over, nil)

	// Center-crop to target dimensions.
	offsetX := (scaledW - w) / 2
	offsetY := (scaledH - h) / 2

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Copy(dst, image.Point{}, scaled, image.Rect(offsetX, offsetY, offsetX+w, offsetY+h), draw.Src, nil)
	return dst
}

// resizeWidth resizes an image to the given width, calculating height automatically
// to preserve aspect ratio.
func resizeWidth(img image.Image, w int) image.Image {
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()

	if srcW <= w {
		return img
	}

	ratio := float64(srcH) / float64(srcW)
	newH := int(float64(w) * ratio)
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
	return dst
}

// encodeImage encodes an image back to the specified format.
func encodeImage(img image.Image, mimeType string, quality int) ([]byte, error) {
	var buf bytes.Buffer

	switch mimeType {
	case "image/jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	case "image/png":
		enc := &png.Encoder{CompressionLevel: png.BestCompression}
		if err := enc.Encode(&buf, img); err != nil {
			return nil, err
		}
	case "image/gif":
		if err := gif.Encode(&buf, img, nil); err != nil {
			return nil, err
		}
	case "image/webp":
		// WebP encoding requires CGO; re-encode as JPEG instead.
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported output format: %s", mimeType)
	}

	return buf.Bytes(), nil
}
