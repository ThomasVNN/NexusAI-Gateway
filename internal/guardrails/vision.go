package guardrails

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"regexp"
	"strings"
)

// ImageProcessor interface for image processing
type ImageProcessor interface {
	Process(image []byte) ([]byte, error)
	DetectPII(image []byte) ([]PIIRegion, error)
}

// PIIRegion represents a region of PII detected in an image
type PIIRegion struct {
	Type      PIIType `json:"type"`
	X, Y      int     `json:"x"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Text      string  `json:"text,omitempty"`
	Confidence float64 `json:"confidence"`
}

// VisionGuardrail implements vision/image-based guardrails
type VisionGuardrail struct {
	mode       GuardrailMode
	processors []ImageProcessor
	blurFaces  bool
	redactText bool
}

// TextDetector is a simple text detection processor
type TextDetector struct {
	patterns []*regexp.Regexp
}

// NewTextDetector creates a new text detector
func NewTextDetector() *TextDetector {
	return &TextDetector{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`\d{3}-\d{2}-\d{4}`), // SSN
			regexp.MustCompile(`\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}`), // Credit card
			regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`), // Email
		},
	}
}

// NewVisionGuardrail creates a new vision guardrail
func NewVisionGuardrail(mode GuardrailMode) *VisionGuardrail {
	return &VisionGuardrail{
		mode:       mode,
		processors: []ImageProcessor{},
		blurFaces:  true,
		redactText: true,
	}
}

// Name returns the guardrail name
func (g *VisionGuardrail) Name() string {
	return "vision-guardrail"
}

// Priority returns the priority (lower = earlier)
func (g *VisionGuardrail) Priority() int {
	return 8
}

// Stage returns the stage this guardrail runs at
func (g *VisionGuardrail) Stage() GuardrailStage {
	return StagePreCall
}

// Check processes images in the request for PII
func (g *VisionGuardrail) Check(ctx context.Context, gc *GuardrailContext) (*GuardrailResult, error) {
	if gc.Request == nil || len(gc.Request.Images) == 0 {
		return &GuardrailResult{
			Passed:  true,
			Action:  ActionAllow,
			Message: "No images to process",
		}, nil
	}

	var detections []*Detection
	var processedImages []ImageData

	for _, img := range gc.Request.Images {
		// Skip URL-based images (would need OCR)
		if img.Type == "url" && img.URL != "" {
			detections = append(detections, &Detection{
				Type:      "vision_url",
				Value:     img.URL,
				Severity:  SeverityLow,
				Confidence: 0.5,
			})
			processedImages = append(processedImages, img)
			continue
		}

		// Process base64 images
		if img.Type == "base64" && img.Data != "" {
			// Note: In production, this would use OCR and face detection
			// For now, we mark as requiring processing
			detections = append(detections, &Detection{
				Type:      "image_base64",
				Value:     "[base64_image_data]",
				Severity:  SeverityLow,
				Confidence: 0.5,
			})
		}
	}

	if len(detections) == 0 {
		return &GuardrailResult{
			Passed:  true,
			Action:  ActionAllow,
			Message: "No vision threats detected",
		}, nil
	}

	result := &GuardrailResult{
		Passed:     true,
		Action:    g.modeToAction(),
		Message:   "Vision content processed",
		Detections: detections,
		Metadata: map[string]interface{}{
			"image_count":   len(gc.Request.Images),
			"blur_faces":    g.blurFaces,
			"redact_text":   g.redactText,
			"processors":    len(g.processors),
		},
	}

	return result, nil
}

// modeToAction converts the guardrail mode to an action
func (g *VisionGuardrail) modeToAction() GuardrailAction {
	switch g.mode {
	case ModeBlock:
		return ActionBlock
	case ModeWarn:
		return ActionWarn
	case ModeLog:
		return ActionLog
	default:
		return ActionWarn
	}
}

// AddProcessor adds an image processor
func (g *VisionGuardrail) AddProcessor(processor ImageProcessor) {
	g.processors = append(g.processors, processor)
}

// SetBlurFaces enables/disables face blurring
func (g *VisionGuardrail) SetBlurFaces(blur bool) {
	g.blurFaces = blur
}

// SetRedactText enables/disables text redaction
func (g *VisionGuardrail) SetRedactText(redact bool) {
	g.redactText = redact
}

// SetMode updates the guardrail mode
func (g *VisionGuardrail) SetMode(mode GuardrailMode) {
	g.mode = mode
}

// BlurRegion applies blur to a specific region of an image
func BlurRegion(img image.Image, x, y, w, h int) image.Image {
	bounds := img.Bounds()
	blurImg := image.NewRGBA(bounds)

	// Simple box blur implementation
	radius := 5
	for py := bounds.Min.Y; py < bounds.Max.Y; py++ {
		for px := bounds.Min.X; px < bounds.Max.X; px++ {
			// Check if pixel is in blur region
			if px >= x && px < x+w && py >= y && py < y+h {
				// Calculate average color in radius
				var rSum, gSum, bSum, count int64
				for dy := -radius; dy <= radius; dy++ {
					for dx := -radius; dx <= radius; dx++ {
						sx := px + dx
						sy := py + dy
						if sx >= bounds.Min.X && sx < bounds.Max.X && sy >= bounds.Min.Y && sy < bounds.Max.Y {
							c := img.At(sx, sy)
							col := color.RGBAModel.Convert(c).(color.RGBA)
							rSum += int64(col.R)
							gSum += int64(col.G)
							bSum += int64(col.B)
							count++
						}
					}
				}
				if count > 0 {
					blurImg.SetRGBA(px, py, color.RGBA{
						R: uint8(rSum / count),
						G: uint8(gSum / count),
						B: uint8(bSum / count),
						A: 255,
					})
				}
			} else {
				blurImg.Set(px, py, img.At(px, py))
			}
		}
	}

	return blurImg
}

// EncodeJPEG encodes an image to JPEG format
func EncodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf strings.Builder
	writer := &strings.Builder{}
	
	// Use a temporary file or bytes.Buffer in production
	// For now, return a placeholder
	_ = writer
	_ = buf
	_ = quality

	// In production, use:
	// var buf bytes.Buffer
	// err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	// return buf.Bytes(), err

	return nil, nil
}

// EncodePNG encodes an image to PNG format
func EncodePNG(img image.Image) ([]byte, error) {
	// In production, use:
	// var buf bytes.Buffer
	// err := png.Encode(&buf, img)
	// return buf.Bytes(), err

	return nil, nil
}

// DecodeImage decodes image data
func DecodeImage(data []byte) (image.Image, string, error) {
	// Try JPEG first
	reader := strings.NewReader(string(data))
	_, _ = reader.Read(data)
	
	// Simple format detection based on magic bytes
	if len(data) >= 4 {
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			img, err := jpeg.Decode(strings.NewReader(string(data)))
			return img, "jpeg", err
		}
		if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
			img, err := png.Decode(strings.NewReader(string(data)))
			return img, "png", err
		}
	}

	return nil, "unknown", nil
}

// OCRText extracts text from image (placeholder for actual OCR integration)
func OCRText(img image.Image) (string, error) {
	// In production, integrate with Tesseract or cloud OCR service
	return "", nil
}

// DetectFaces detects faces in an image (placeholder for actual face detection)
func DetectFaces(img image.Image) ([]image.Rectangle, error) {
	// In production, integrate with OpenCV or cloud face detection
	return nil, nil
}

// RedactSensitiveRegions blurs or redacts detected sensitive regions
func RedactSensitiveRegions(img image.Image, regions []PIIRegion) image.Image {
	result := img

	for _, region := range regions {
		bounds := image.Rect(
			region.X, region.Y,
			region.X+region.Width, region.Y+region.Height,
		)
		result = BlurRegion(result, bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy())
	}

	return result
}
