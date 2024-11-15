package handler

import (
    "errors"
    "log"
    "net/http"

    "github.com/go-playground/validator/v10"
    "github.com/goccy/go-json"
    "github.com/yemyoaung/managing-vehicle-tracking-common"
    "github.com/yemyoaung/managing-vehicle-tracking-tracking-svc/internal/services"
)

var (
    ErrMethodNotAllowed = errors.New("method was not allowed")
    ErrNotFound         = errors.New("not found")
)

type V1TrackingHandler struct {
    trackingService services.TrackingService
    validate        *validator.Validate
}

func NewV1TrackingHandler(vehicleService services.TrackingService, validate *validator.Validate) *V1TrackingHandler {
    return &V1TrackingHandler{trackingService: vehicleService, validate: validate}
}

func (h *V1TrackingHandler) methodWasNotAllowed(w http.ResponseWriter) {
    common.HandleError(http.StatusMethodNotAllowed, w, ErrMethodNotAllowed)
}

func (h *V1TrackingHandler) FindTrackingData(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        h.methodWasNotAllowed(w)
        return
    }
    vehicles, err := h.trackingService.FindTrackingData(r.Context(), r.URL.Query())
    if err != nil {
        common.HandleError(http.StatusBadRequest, w, err)
        return
    }

    if len(vehicles) == 0 {
        common.HandleError(http.StatusNotFound, w, ErrNotFound)
        return
    }

    if err = json.NewEncoder(w).Encode(
        common.DefaultSuccessResponse(
            vehicles,
            "successfully fetched tracking data",
        ),
    ); err != nil {
        log.Printf("Failed to encode response: %v", err)
    }
}
