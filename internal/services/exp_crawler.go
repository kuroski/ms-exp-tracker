package services

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/disintegration/imaging"
	"github.com/kbinani/screenshot"
	"github.com/otiai10/gosseract/v2"
)

const ContentAreaHeight = 70

type CrawlResult struct {
	Exp        int
	Percentage float64
}

type ExpCrawler interface {
	Crawl() (result CrawlResult, err error)
}

type ScreenExpCrawler struct {
	window *application.WebviewWindow
}

func NewScreenExpCrawler(window *application.WebviewWindow) *ScreenExpCrawler {
	return &ScreenExpCrawler{
		window: window,
	}
}

func (s *ScreenExpCrawler) Crawl() (result CrawlResult, err error) {
	img, err := s.captureScreen()
	if err != nil {
		return CrawlResult{}, err
	}

	simg, _, _ := image.Decode(bytes.NewReader(img))
	f, _ := os.Create(fmt.Sprintf(".tmp/%d_crawl.jpeg", time.Now().Unix()))
	defer f.Close()
	jpeg.Encode(f, simg, &jpeg.Options{Quality: 90})

	return s.extractXPFromImage(img)
}

func (s *ScreenExpCrawler) captureScreen() (img []byte, err error) {
	posX, posY := s.window.RelativePosition()
	width, height := s.window.Size()
	rect := image.Rect(posX, posY+ContentAreaHeight, posX+width, posY+height)
	expBarImg, err := screenshot.CaptureRect(rect)
	if err != nil {
		return nil, err
	}

	processedImg := s.preprocessImage(expBarImg)

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, processedImg, &jpeg.Options{Quality: 95})
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *ScreenExpCrawler) preprocessImage(img image.Image) image.Image {
	grayImg := imaging.Grayscale(img)

	binarized := imaging.AdjustFunc(grayImg, func(c color.NRGBA) color.NRGBA {
		threshold := uint8(170) // Adjust threshold as needed (0-255)
		if c.R > threshold {    // Since it's grayscale, R=G=B
			return color.NRGBA{255, 255, 255, 255}
		}
		return color.NRGBA{0, 0, 0, 255}
	})

	resized := imaging.Resize(binarized, binarized.Bounds().Dx()*2, binarized.Bounds().Dy()*2, imaging.Linear)

	return resized
}

func (s *ScreenExpCrawler) extractXPFromImage(img []byte) (result CrawlResult, err error) {
	client := gosseract.NewClient()
	defer client.Close()

	if err := client.SetImageFromBytes(img); err != nil {
		return CrawlResult{}, err
	}
	if err := client.SetWhitelist("0123456789%.["); err != nil {
		return CrawlResult{}, err
	}

	client.SetPageSegMode(gosseract.PSM_SINGLE_LINE)

	var text string

	// try extracting the text 3 times
	for range 3 {
		text, err = client.Text()
		if err == nil && text != "" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	if text == "" {
		return CrawlResult{}, err
	}

	// regex pattern: Match "<number>[<number>%"
	re := regexp.MustCompile(`(\d+)\[(\d+\.\d+)%`)
	matches := re.FindStringSubmatch(text)
	if len(matches) < 3 {
		return CrawlResult{}, fmt.Errorf("no match found in text: %s", text)
	}

	xp, err := strconv.Atoi(matches[1])
	if err != nil {
		return CrawlResult{}, fmt.Errorf("invalid XP format: %w", err)
	}

	percent, err := strconv.ParseFloat(matches[2], 64)
	if err != nil {
		return CrawlResult{}, fmt.Errorf("invalid percentage format: %w", err)
	}

	return CrawlResult{
		Exp:        xp,
		Percentage: percent,
	}, nil
}
