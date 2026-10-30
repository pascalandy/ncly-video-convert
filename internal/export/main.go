// Command export runs repository tasks through the SDK.
// The import name ext never collides with the domain, such as fmt.
package main

import (
	"github.com/pascalandy/nicely/sdk/extensionrepo"

	ext "github.com/pascalandy/ncly-video-convert"
)

func main() {
	extensionrepo.Main(ext.Extension)
}
