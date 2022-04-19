package projects

import (
	"github.com/maltejk/metakube-go-client/pkg/client/project"
)

// IsNotFound returns whether the given error is of type NotFound or not.
func IsNotFound(err error) bool {
	// 404 NotFound is hidden in GetProjectDefault and only known by calling Code()
	if status := err.(*project.GetProjectDefault).Code(); status == 404 {
		return true
	}
	return false
}
