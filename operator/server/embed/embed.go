package embed

import "embed"

// UIAssets contains the embedded UI files from ui/dist.
// When the UI hasn't been built yet, this will be empty.
//
//go:embed all:dist
var UIAssets embed.FS
