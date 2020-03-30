package elasticsearch

import (
	"github.com/landoop/lenses-go/pkg/api"
)

// IndexView type
type IndexView struct {
	api.Index         `header:"inline"`
	AvailableReplicas int `header:"Available replicas"`
}

// MakeIndexView creates a presentation for later consumption
func MakeIndexView(esIndex api.Index) IndexView {
	availableReplicas := api.GetAvailableReplicas(esIndex)

	esIndex.ShardsCount = len(esIndex.Shards)
	view := IndexView{esIndex, availableReplicas}

	return view
}
