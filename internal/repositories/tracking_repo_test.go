package repositories

import (
    "context"
    "fmt"
    "log"
    "math/rand"
    "testing"

    "github.com/yemyoaung/managing-vehicle-tracking-models"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

const (
    connStr = "mongodb://yoma_fleet:YomaFleet!123@localhost:27017"
)

func getTrackingRepo() (*mongo.Client, *MongoTackingRepository, error) {
    // we can also use mock database for testing
    // but for now we will use real database to make sure everything is working fine
    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(connStr))
    if err != nil {
        return nil, nil, err
    }

    repo := NewMongoTackingRepository(client.Database("tracking"))

    return client, repo, nil
}

var VehicleStatuses = []models.VehicleStatus{
    models.VehicleStatusActive,
    models.VehicleStatusInactive,
    models.VehicleStatusRepair,
    models.VehicleStatusSold,
    models.VehicleStatusRented,
}

var FuelConditions = []models.FuelCondition{
    models.FuelConditionEmpty,
    models.FuelConditionLow,
    models.FuelConditionHalf,
    models.FuelConditionFull,
}

func getRandomTrackingData() (*models.TrackingData, error) {
    trackingData, err := models.NewTrackingData().SetVehicleID(
        fmt.Sprintf("%d735cc0f1af72af5f7cdcdee", rand.Intn(9)),
    )
    if err != nil {
        return nil, err
    }
    trackingData.SetLocation(fmt.Sprintf("Location %d", rand.Intn(9))).
        SetMileage(rand.Float64() * 1000).
        SetStatus(VehicleStatuses[rand.Intn(len(VehicleStatuses))]).
        SetFuelCondition(FuelConditions[rand.Intn(len(FuelConditions))])
    if err := trackingData.Build(); err != nil {
        return nil, err
    }
    return trackingData, nil
}

func TestMongoTackingRepository_CreateTrackingData(t *testing.T) {
    client, repo, err := getTrackingRepo()

    if err != nil {
        t.Fatal(err)
    }

    defer func(client *mongo.Client, ctx context.Context) {
        err := client.Disconnect(ctx)
        if err != nil {
            log.Println("Failed to disconnect from database")
        }
    }(client, context.Background())

    trackingData, err := getRandomTrackingData()

    if err != nil {
        t.Fatal(err)
    }

    err = repo.CreateTrackingData(context.Background(), trackingData)

    if err != nil {
        t.Fatal(err)
    }

    if trackingData.ID.IsZero() {
        t.Fatal("ID should not be zero")
    }
}

func TestMongoTrackingRepository_FindTrackingData(t *testing.T) {
    client, repo, err := getTrackingRepo()

    if err != nil {
        t.Fatal(err)
    }

    defer func(client *mongo.Client, ctx context.Context) {
        err := client.Disconnect(ctx)
        if err != nil {
            log.Println("Failed to disconnect from database")
        }
    }(client, context.Background())

    for i := 0; i < 10; i++ {
        trackingData, err := getRandomTrackingData()
        if err != nil {
            t.Fatal(err)
        }
        if err := repo.CreateTrackingData(context.Background(), trackingData); err != nil {
            t.Fatal(err)
        }
    }

    for i := 1; i <= 5; i++ {
        trackingData, err := repo.FindTrackingData(
            context.Background(), &TrackingFilter{
                Page:     i,
                PageSize: 2,
            },
        )
        if err != nil {
            t.Fatal(err)
        }
        if len(trackingData) != 2 {
            t.Fatal("Should return 2 tracking data")
        }
    }

    trackingData, err := repo.FindTrackingData(
        context.Background(), &TrackingFilter{
            Page:      1,
            PageSize:  10,
            SortField: "location",
        },
    )

    if err != nil {
        t.Fatal(err)
    }

    if len(trackingData) != 10 {
        t.Fatal("Should return 10 tracking data")
    }

    for i := 0; i < len(trackingData)-1; i++ {
        if trackingData[i].Location > trackingData[i+1].Location {
            t.Fatal("Tracking data should be sorted")
        }
    }

    trackingData, err = repo.FindTrackingData(
        context.Background(), &TrackingFilter{
            Page:      1,
            PageSize:  10,
            SortField: "location",
            SortOrder: "desc",
        },
    )

    if err != nil {
        t.Fatal(err)
    }

    if len(trackingData) != 10 {
        t.Fatal("Should return 10 tracking data")
    }

    for i := 0; i < len(trackingData)-1; i++ {
        if trackingData[i].Location < trackingData[i+1].Location {
            t.Fatal("Tracking data should be sorted")
        }
    }

    trackingData, err = repo.FindTrackingData(
        context.Background(), &TrackingFilter{
            Page:      1,
            PageSize:  10,
            VehicleID: trackingData[0].VehicleID.Hex(),
        },
    )

    if err != nil {
        t.Fatal(err)
    }

    for _, data := range trackingData {
        if data.VehicleID.Hex() != trackingData[0].VehicleID.Hex() {
            t.Fatal("tracking data name should be " + trackingData[0].VehicleID.Hex())
        }
    }
}
