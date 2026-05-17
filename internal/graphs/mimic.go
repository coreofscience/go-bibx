package graphs

import "github.com/hmdsefi/gograph"

func Mimic[K comparable](g gograph.Graph[K], overrideOptions ...gograph.GraphOptionFunc) gograph.Graph[K] {
	options := make([]gograph.GraphOptionFunc, 0)
	if g.IsDirected() {
		options = append(options, gograph.Directed())
	}
	if g.IsWeighted() {
		options = append(options, gograph.Weighted())
	}
	if g.IsAcyclic() {
		options = append(options, gograph.Acyclic())
	}
	options = append(options, overrideOptions...)
	return gograph.New[K](options...)
}
