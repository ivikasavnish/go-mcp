package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// Browser manages browser automation
type Browser struct {
	config  *BrowserConfig
	browser *rod.Browser
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewBrowser creates a new browser instance
func NewBrowser(config *BrowserConfig) *Browser {
	ctx, cancel := context.WithCancel(context.Background())
	return &Browser{
		config: config,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start initializes and starts the browser
func (b *Browser) Start() error {
	url := launcher.New().
		Headless(b.config.Headless).
		Proxy(b.config.Proxy).
		MustLaunch()

	browser := rod.New().ControlURL(url).MustConnect()

	if b.config.UserAgent != "" {
		browser = browser.MustIncognito()
	}

	b.browser = browser
	return nil
}

// Stop closes the browser
func (b *Browser) Stop() error {
	if b.browser != nil {
		b.browser.MustClose()
	}
	b.cancel()
	return nil
}

// Navigate navigates to a URL and returns the result
func (b *Browser) Navigate(url string) (*NavigationResult, error) {
	start := time.Now()

	page := b.browser.MustPage(url)
	if b.config.UserAgent != "" {
		page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: b.config.UserAgent,
		})
	}

	if b.config.ViewPort != nil {
		//page.MustSetViewport(b.config.ViewPort.Width, b.config.ViewPort.Height)
	}

	for key, value := range b.config.Headers {
		_, _ = key, value
		//page.MustSetExtraHeaders(map[string]string{key: value})
	}

	// Wait for network idle
	page.MustWaitNavigation()

	result := &NavigationResult{
		Success:  true,
		URL:      page.MustInfo().URL,
		Title:    page.MustInfo().Title,
		LoadTime: time.Since(start).Seconds(),
	}

	return result, nil
}

// ExecuteSequence executes an automation sequence
func (b *Browser) ExecuteSequence(seq *AutomationSequence) error {
	page := b.browser.MustPage("")

	for _, step := range seq.Steps {
		if err := b.executeStep(page, step); err != nil {
			return fmt.Errorf("step %s failed: %w", step.Type, err)
		}

		if step.Wait > 0 {
			time.Sleep(step.Wait)
		}
	}

	return nil
}

// executeStep executes a single automation step
func (b *Browser) executeStep(page *rod.Page, step AutomationStep) error {
	ctx := b.ctx
	if step.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, step.Timeout)
		defer cancel()
	}

	switch step.Type {
	case "navigate":
		url, ok := step.Params["url"].(string)
		if !ok {
			return fmt.Errorf("invalid url parameter")
		}
		page.MustNavigate(url).MustWaitNavigation()

	case "click":
		selector, ok := step.Params["selector"].(string)
		if !ok {
			return fmt.Errorf("invalid selector parameter")
		}
		page.MustElement(selector).MustClick()

	case "type":
		selector, ok := step.Params["selector"].(string)
		if !ok {
			return fmt.Errorf("invalid selector parameter")
		}
		text, ok := step.Params["text"].(string)
		if !ok {
			return fmt.Errorf("invalid text parameter")
		}
		page.MustElement(selector).MustInput(text)

	case "screenshot":
		format, _ := step.Params["format"].(string)
		if format == "" {
			format = "png"
		}
		fullPage, _ := step.Params["full_page"].(bool)

		var _ []byte
		if fullPage {
			_ = page.MustScreenshotFullPage()
		} else {
			_ = page.MustScreenshot()
		}

		// Store screenshot in context or return it

	case "scrape":
		selector, ok := step.Params["selector"].(string)
		if !ok {
			return fmt.Errorf("invalid selector parameter")
		}
		elements := page.MustElements(selector)
		_ = elements
		// Process scraped elements

	default:
		return fmt.Errorf("unknown step type: %s", step.Type)
	}

	return nil
}

// Scrape extracts data from the current page using selectors
func (b *Browser) Scrape(selectors map[string]string) (*ScrapingResult, error) {
	page := b.browser.MustPage("")
	result := &ScrapingResult{
		URL:       page.MustInfo().URL,
		Data:      make(map[string]interface{}),
		Timestamp: time.Now(),
	}

	for key, selector := range selectors {
		elements := page.MustElements(selector)
		if len(elements) == 1 {
			result.Data[key] = elements[0].MustText()
		} else {
			texts := make([]string, len(elements))
			for i, el := range elements {
				texts[i] = el.MustText()
			}
			result.Data[key] = texts
		}
	}

	return result, nil
}

// CaptureScreenshot takes a screenshot of the current page
func (b *Browser) CaptureScreenshot(fullPage bool) (*Screenshot, error) {
	page := b.browser.MustPage("")

	var buf []byte
	if fullPage {
		buf = page.MustScreenshotFullPage()
	} else {
		buf = page.MustScreenshot()
	}

	return &Screenshot{
		Data:      buf,
		Format:    "png",
		FullPage:  fullPage,
		Timestamp: time.Now(),
	}, nil
}
