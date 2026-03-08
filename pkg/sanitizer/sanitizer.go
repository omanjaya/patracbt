package sanitizer

import "github.com/microcosm-cc/bluemonday"

var policy *bluemonday.Policy

func init() {
	policy = bluemonday.UGCPolicy()
	// Allow tables
	policy.AllowElements("table", "thead", "tbody", "tr", "th", "td")
	policy.AllowAttrs("style", "class", "border", "cellpadding", "cellspacing").OnElements("table", "th", "td")
	// Allow images
	policy.AllowImages()
	policy.AllowAttrs("src", "alt", "width", "height", "style").OnElements("img")
	// Allow basic formatting
	policy.AllowAttrs("style", "class").Globally()
	// Allow audio/video for question media
	policy.AllowElements("audio", "source", "video")
	policy.AllowAttrs("src", "type", "controls").OnElements("audio", "source", "video")
}

func SanitizeHTML(input string) string {
	return policy.Sanitize(input)
}
