package repositories

import (
    "context"
    "errors"
    "fmt"
    "log"

    "github.com/yemyoaung/managing-vehicle-tracking-models"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var (
    ErrInvalidID = errors.New("invalid id")
)

type TrackingFilter struct {
    Page          int                  `json:"page"`
    PageSize      int                  `json:"limit"`
    SortField     string               `json:"sort_by"`
    SortOrder     string               `json:"sort_order"`
    VehicleID     string               `json:"vehicle_id"`
    Location      string               `json:"location"`
    Mileage       float64              `json:"mileage"`
    Status        models.VehicleStatus `json:"status"`
    FuelCondition models.FuelCondition `json:"fuel_condition"`

    vehicleID primitive.ObjectID
}

func (t *TrackingFilter) VehicleObjID() primitive.ObjectID {
    return t.vehicleID
}

func (t *TrackingFilter) Build() error {
    if t.Page == 0 {
        t.Page = 1
    }
    if t.PageSize == 0 {
        t.PageSize = 10
    }
    if t.PageSize > 100 {
        t.PageSize = 100
    }
    if t.SortField == "" {
        t.SortField = "created_at"
    }
    if t.SortOrder == "" {
        t.SortOrder = "asc"
    }
    if t.VehicleID != "" {
        id, err := primitive.ObjectIDFromHex(t.VehicleID)
        if err != nil {
            return ErrInvalidID
        }
        t.vehicleID = id
    }
    if t.Status != "" {
        if err := t.Status.Valid(); err != nil {
            return err
        }
    }
    if t.FuelCondition != "" {
        if err := t.FuelCondition.Valid(); err != nil {
            return err
        }
    }
    return nil
}

type TrackingRepository interface {
    CreateTrackingData(ctx context.Context, trackingData *models.TrackingData) error
    FindTrackingData(ctx context.Context, filter *TrackingFilter) ([]*models.TrackingData, error)
}

type MongoTackingRepository struct {
    collection *mongo.Collection
}

func NewMongoTackingRepository(db *mongo.Database) *MongoTackingRepository {
    trackingCollection := db.Collection("tracking")
    return &MongoTackingRepository{
        collection: trackingCollection,
    }
}

func (repo *MongoTackingRepository) CreateTrackingData(ctx context.Context, trackingData *models.TrackingData) error {
    if err := trackingData.Build(); err != nil {
        return err
    }
    result, err := repo.collection.InsertOne(ctx, trackingData)
    if err != nil {
        return err
    }
    trackingData.ID = result.InsertedID.(primitive.ObjectID)
    return nil
}

func (repo *MongoTackingRepository) FindTrackingData(
    ctx context.Context,
    filter *TrackingFilter,
) ([]*models.TrackingData, error) {
    var trackingData []*models.TrackingData
    bsonMFilter := bson.M{}
    findOptions := options.Find()
    if filter != nil {
        if err := filter.Build(); err != nil {
            return nil, err
        }
        if filter.VehicleID != "" {
            bsonMFilter["vehicle_id"] = filter.VehicleObjID()
        }
        if filter.Location != "" {
            bsonMFilter["location"] = bson.M{"$regex": fmt.Sprintf("^%s", filter.Location), "$options": "i"}
        }
        if filter.Mileage != 0 {
            bsonMFilter["mileage"] = bson.M{"$gte": filter.Mileage}
        }
        if filter.Status != "" {
            bsonMFilter["status"] = filter.Status
        }
        if filter.FuelCondition != "" {
            bsonMFilter["fuel_condition"] = filter.FuelCondition
        }
        if filter.SortField != "" {
            order := 1
            if filter.SortOrder == "desc" {
                order = -1
            }
            findOptions.SetSort(bson.D{{Key: filter.SortField, Value: order}})
        }
        findOptions.SetSkip(int64((filter.Page - 1) * filter.PageSize))
        findOptions.SetLimit(int64(filter.PageSize))
    }
    cursor, err := repo.collection.Find(ctx, bsonMFilter, findOptions)
    if err != nil {
        return nil, err
    }
    defer func(cursor *mongo.Cursor, ctx context.Context) {
        err := cursor.Close(ctx)
        if err != nil {
            log.Println("Error closing cursor", err)
        }
    }(cursor, ctx)
    for cursor.Next(ctx) {
        var data models.TrackingData
        if err := cursor.Decode(&data); err != nil {
            return nil, err
        }
        trackingData = append(trackingData, &data)
    }
    return trackingData, nil
}
