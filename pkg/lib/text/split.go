package text

import "strings"

func Split(str, sep string) ([]string, error) {
	words := strings.Split(str, sep)

	return words, nil
}
