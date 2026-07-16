// Package uaparse provides a small, dependency-free User-Agent parser good
// enough for surfacing the browser and operating system of a client to a
// help-desk operator. It intentionally favours readability over exhaustive
// coverage of every exotic UA string.
package uaparse

import "strings"

// Result holds the parsed browser and OS details.
type Result struct {
	Browser string `json:"browser"`
	OS      string `json:"os"`
	Mobile  bool   `json:"mobile"`
}

// Parse extracts browser and OS information from a User-Agent header value.
func Parse(ua string) Result {
	return Result{
		Browser: browser(ua),
		OS:      os(ua),
		Mobile:  strings.Contains(ua, "Mobi") || strings.Contains(ua, "Android"),
	}
}

func browser(ua string) string {
	switch {
	case ua == "":
		return "Unknown"
	case strings.Contains(ua, "Edg/") || strings.Contains(ua, "Edge/"):
		return "Microsoft Edge"
	case strings.Contains(ua, "OPR/") || strings.Contains(ua, "Opera"):
		return "Opera"
	case strings.Contains(ua, "Firefox/"):
		return "Firefox"
	case strings.Contains(ua, "Chrome/"):
		return "Chrome"
	case strings.Contains(ua, "Safari/"):
		return "Safari"
	case strings.Contains(ua, "curl/"):
		return "curl"
	case strings.Contains(ua, "Wget/"):
		return "Wget"
	default:
		return "Unknown"
	}
}

func os(ua string) string {
	switch {
	case strings.Contains(ua, "Windows NT 10.0"):
		return "Windows 10/11"
	case strings.Contains(ua, "Windows NT 6.3"):
		return "Windows 8.1"
	case strings.Contains(ua, "Windows"):
		return "Windows"
	case strings.Contains(ua, "Android"):
		return "Android"
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad"):
		return "iOS"
	case strings.Contains(ua, "Mac OS X") || strings.Contains(ua, "Macintosh"):
		return "macOS"
	case strings.Contains(ua, "CrOS"):
		return "ChromeOS"
	case strings.Contains(ua, "Linux"):
		return "Linux"
	default:
		return "Unknown"
	}
}
