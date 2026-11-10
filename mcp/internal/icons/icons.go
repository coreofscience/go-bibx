// Package icons holds the bibx logo rendered at several sizes, ready to be
// advertised by the MCP server as data URIs.
//
// The PNGs are rendered from the SVGs in the assets folder, see
// assets/icons for the full set of sizes.
package icons

import (
	"embed"
	"encoding/base64"
	"fmt"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed *.png
var files embed.FS

var sizes = []int{48, 96, 192}

var themes = []struct {
	name  string
	theme sdk.IconTheme
}{
	{"light", sdk.IconThemeLight},
	{"dark", sdk.IconThemeDark},
}

// All returns the bibx icons for every theme and size.
func All() []sdk.Icon {
	icons := make([]sdk.Icon, 0, len(sizes)*len(themes))
	for _, t := range themes {
		for _, size := range sizes {
			name := fmt.Sprintf("bibx-%s-%d.png", t.name, size)
			data, err := files.ReadFile(name)
			if err != nil {
				// The files are embedded at build time, so this is a bug.
				panic(err)
			}
			icons = append(icons, sdk.Icon{
				Source:   "data:image/png;base64," + base64.StdEncoding.EncodeToString(data),
				MIMEType: "image/png",
				Sizes:    []string{fmt.Sprintf("%dx%d", size, size)},
				Theme:    t.theme,
			})
		}
	}
	return icons
}
