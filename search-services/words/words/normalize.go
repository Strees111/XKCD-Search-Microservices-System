package words

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/kljensen/snowball/english"
)

func Normalize(phrase string) []string {
	phrase = strings.ToLower(phrase)
	re := regexp.MustCompile(`[a-z0-9]+`)
	words := re.FindAllString(phrase, -1)
	unique := make(map[string]struct{})
	for _, v := range words {
		if english.IsStopWord(v) {
			continue
		}
		norm := english.Stem(v, false)
		unique[norm] = struct{}{}
	}

	return slices.Collect(maps.Keys(unique))
}
