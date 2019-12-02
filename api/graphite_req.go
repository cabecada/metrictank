package api

import (
	"github.com/grafana/metrictank/api/models"
	"github.com/grafana/metrictank/expr"
)

// ReqMap is a map of requests of data,
// it has single requests for which no pre-normalization effort will be performed, and
// requests are that can be pre-normalized together to the same resolution, bundled by their PNGroup
type ReqMap struct {
	single   []models.Req
	pngroups map[expr.PNGroup][]models.Req
	cnt      uint32
}

func NewReqMap() *ReqMap {
	return &ReqMap{
		pngroups: make(map[expr.PNGroup][]models.Req),
	}
}

func (r *ReqMap) Add(req models.Req, group expr.PNGroup) {
	r.cnt++
	if group == 0 {
		r.single = append(r.single, req)
	}
	r.pngroups[group] = append(r.pngroups[group], req)
}
