// Package messages identifies application errors without altering raw diagnostics.
package messages

import (
	_ "embed"
	"encoding/json"
	"regexp"
	"strconv"
)

//go:embed errors.json
var catalogue []byte

type rule struct {
	Code       string `json:"code"`
	Pattern    string `json:"pattern"`
	expression *regexp.Regexp
}

var rules = func() []rule {
	var result []rule
	if err := json.Unmarshal(catalogue, &result); err != nil {
		panic(err)
	}
	for i := range result {
		result[i].expression = regexp.MustCompile(result[i].Pattern)
	}
	return result
}()

func Identify(message string) (string, map[string]string) {
	for _, rule := range rules {
		parts := rule.expression.FindStringSubmatch(message)
		if parts == nil {
			continue
		}
		params := map[string]string{}
		for i, value := range parts[1:] {
			params["p"+strconv.Itoa(i)] = value
		}
		return rule.Code, params
	}
	return "", nil
}
