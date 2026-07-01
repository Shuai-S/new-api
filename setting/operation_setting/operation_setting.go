package operation_setting

import "strings"

var DemoSiteEnabled = false
var SelfUseModeEnabled = false

var AutomaticDisableKeywords = []string{
	"Your credit balance is too low",
	"This organization has been disabled.",
	"You exceeded your current quota",
	"Permission denied",
	"The security token included in the request is invalid",
	"Operation not allowed",
	"Your account is not authorized",
}

var AutomaticRetryKeywords []string

func AutomaticDisableKeywordsToString() string {
	return strings.Join(AutomaticDisableKeywords, "\n")
}

func AutomaticDisableKeywordsFromString(s string) {
	AutomaticDisableKeywords = keywordsFromString(s)
}

func AutomaticRetryKeywordsToString() string {
	return strings.Join(AutomaticRetryKeywords, "\n")
}

func AutomaticRetryKeywordsFromString(s string) {
	AutomaticRetryKeywords = keywordsFromString(s)
}

func ShouldRetryByKeyword(message string) bool {
	return matchesKeywords(message, AutomaticRetryKeywords)
}

func matchesKeywords(message string, keywords []string) bool {
	message = strings.ToLower(message)
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

func keywordsFromString(s string) []string {
	keywords := []string{}
	for _, keyword := range strings.Split(s, "\n") {
		keyword = strings.TrimSpace(keyword)
		keyword = strings.ToLower(keyword)
		if keyword != "" {
			keywords = append(keywords, keyword)
		}
	}
	return keywords
}
