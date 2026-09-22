package main

import (
	"log"
	"os"
)

var defaultLanguage = "en"

func validLanguage(language string) bool {
	return language == "en" || language == "zh-CN"
}

func configureLanguage() {
	defaultLanguage = os.Getenv("DEFAULT_LANGUAGE")
	if defaultLanguage == "" {
		defaultLanguage = "en"
	} else if !validLanguage(defaultLanguage) {
		log.Printf("unsupported DEFAULT_LANGUAGE %q; using en", defaultLanguage)
		defaultLanguage = "en"
	}
}

func effectiveLanguage(language string) string {
	if validLanguage(language) {
		return language
	}
	return defaultLanguage
}
