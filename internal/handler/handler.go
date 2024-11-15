package handler

import "net/http"

type TrackingHandler interface {
    FindTrackingData(w http.ResponseWriter, r *http.Request)
}
