//go:build ompig_strip_word_dictionaries

package wordsegmenter

import "github.com/tmih06/ompig/internal/pigstrip"

// ompig additive (D92): a Ompiglet Binary built with ompig_strip_word_dictionaries embeds no word dictionary. Each run of CJK, Thai, Lao, Khmer or Burmese text is one word, as ICU segments it without dictionary data.

func init() { pigstrip.Strip(pigstrip.ListFeatures, pigstrip.WordDictionaries) }

func cjkBoundaries(string) []int { return nil }

func seaDivide(string, rune, func(int)) {}
