/*
Wait for the new filesystem API planned by a Go proposal (from the core team).

This is currentyl just a stand-in for a real implementation.
That means: it's a quick hack!

It grows whenever I add some functionality needed for tests. Everything here
is quick and dirty right now.
*/
package locate

import (
	"context"

	"github.com/npillmayer/fontfind"
)

type FontLocator func(fontfind.Descriptor) (fontfind.ScalableFont, error)

// FontLocatorWithContext is a context-aware variant of FontLocator.
// Implementations should respect cancellation/deadlines of ctx if possible.
type FontLocatorWithContext func(context.Context, fontfind.Descriptor) (fontfind.ScalableFont, error)
