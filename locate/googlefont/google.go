package googlefont

import (
	"github.com/npillmayer/fontfind"
	"github.com/npillmayer/fontfind/locate"
	"github.com/npillmayer/schuko"
	"github.com/npillmayer/schuko/schukonf/testconfig"
	"github.com/npillmayer/schuko/tracing"
)

// tracer writes to trace with key 'tyse.font'
func tracer() tracing.Trace {
	return tracing.Select("tyse.font")
}

var USE_SYSTEM_IO IO = nil

// Find creates a FontLocator for Google Fonts using default host I/O.
// hostio may be nil (USE_SYSTEM_IO) to use the OS-backed default implementation.
func Find(conf schuko.Configuration, hostio IO) locate.FontLocator {
	svc := newGoogleService(hostio)
	return func(descr fontfind.Descriptor) (fontfind.ScalableFont, error) {
		pattern := descr.Pattern
		style := descr.Style
		weight := descr.Weight
		return svc.findGoogleFont(conf, pattern, style, weight)
	}
}

// SimpleConfig returns a minimal configuration containing only "app-key".
func SimpleConfig(appkey string) schuko.Configuration {
	conf := make(testconfig.Conf)
	conf.Set("app-key", appkey)
	return conf
}
