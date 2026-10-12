package utils

import "strings"

// DisplayName is the one place that decides what name is shown for a person. Every meme API (feed,
// My Memes, and any other) calls it with the user's profile name, so the same creator always appears
// under the same name. Spaces at the ends and repeated spaces inside are cleaned up, and nil is
// returned when the person has no name, so callers can fall back to their own placeholder.
func DisplayName(name *string) *string {
	if name == nil {
		return nil
	}
	cleaned := strings.Join(strings.Fields(*name), " ")
	if cleaned == "" {
		return nil
	}
	return &cleaned
}
