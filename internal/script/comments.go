package script

import "regexp"

var htmlCommentRe = regexp.MustCompile(`<!--[\s\S]*?-->`)

// StripComments removes HTML comments from text.
func StripComments(text string) string {
	return htmlCommentRe.ReplaceAllString(text, "")
}
