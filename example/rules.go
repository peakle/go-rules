package example

import (
	"github.com/quasilyte/go-ruleguard/dsl"

	"github.com/peakle/go-rules"
)

func init() {
	dsl.ImportRules("", rules.Bundle)
}
