// Command ncly-video-convert runs the video-convert extension of Nicely. Its main
// starts nothing before extension.Main, so --help and --version stay safe.
// The import name ext never collides with the domain, such as fmt.
package main

import (
	"github.com/pascalandy/nicely/sdk/extension"

	ext "github.com/pascalandy/ncly-video-convert"
)

func main() { extension.Main(ext.Extension) }
