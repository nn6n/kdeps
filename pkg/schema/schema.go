package schema

import (
	"fmt"
	"kdeps/pkg/utils"
	"os"
	"sync"
)

var (
	cachedVersion    string
	once             sync.Once
	specifiedVersion string = "0.1.46" // Default specified version
	UseLatest        bool   = false
)

// SchemaVersion fetches and returns the schema version based on the cmd.Latest flag.
func SchemaVersion() string {
	if UseLatest { // Reference the global Latest flag from cmd package
		once.Do(func() {
			var err error
			cachedVersion, err = utils.GitHubReleaseFetcher("kdeps/schema", "")
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: Unable to fetch the latest schema version for 'kdeps/schema': %v\n", err)
				os.Exit(1)
			}
		})
		return cachedVersion
	}

	// Use the specified version if not using the latest
	return specifiedVersion
}
