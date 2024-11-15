package services

import (
    "context"
    "net/url"
    "strconv"

    "github.com/goccy/go-json"
    "github.com/yemyoaung/managing-vehicle-tracking-models"
    "github.com/yemyoaung/managing-vehicle-tracking-tracking-svc/internal/repositories"
)

type TrackingService interface {
    TrackVehicle(ctx context.Context, req *models.TrackingDataRequest) error
    FindTrackingData(ctx context.Context, query url.Values) ([]*models.TrackingData, error)
}

type MongoTrackingService struct {
    trackingRepo repositories.TrackingRepository
}

func NewMongoTrackingService(trackingRepo repositories.TrackingRepository) *MongoTrackingService {
    return &MongoTrackingService{
        trackingRepo: trackingRepo,
    }
}

func (s *MongoTrackingService) TrackVehicle(ctx context.Context, req *models.TrackingDataRequest) error {
    err := req.Validate()
    if err != nil {
        return err
    }
    trackingData, err := req.ToTrackingData()
    if err != nil {
        return err
    }
    err = s.trackingRepo.CreateTrackingData(ctx, trackingData)
    if err != nil {
        return err
    }

    return nil
}

func (s *MongoTrackingService) FindTrackingData(ctx context.Context, query url.Values) ([]*models.TrackingData, error) {
    // by converting url.Values to map[string]any and unmarshalling it to TrackingFilter,
    // we can ignore unsupported query parameters
    data := map[string]any{}
    for key, value := range query {
        if key == "page" || key == "limit" {
            converted, err := strconv.Atoi(value[0])
            if err != nil {
                return nil, err
            }
            data[key] = converted
            continue
        }
        if key == "mileage" {
            converted, err := strconv.ParseFloat(value[0], 64)
            if err != nil {
                return nil, err
            }
            data[key] = converted
            continue
        }
        data[key] = value[0]
    }

    buf, err := json.Marshal(data)
    if err != nil {
        return nil, err
    }

    var filter repositories.TrackingFilter
    if err := json.Unmarshal(buf, &filter); err != nil {
        return nil, err
    }

    return s.trackingRepo.FindTrackingData(ctx, &filter)
}
