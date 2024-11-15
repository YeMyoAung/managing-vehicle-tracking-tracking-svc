package app

import (
    "context"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"

    "github.com/go-playground/validator/v10"
    "github.com/goccy/go-json"
    amqp "github.com/rabbitmq/amqp091-go"
    "github.com/yemyoaung/managing-vehicle-tracking-common"
    "github.com/yemyoaung/managing-vehicle-tracking-models"
    "github.com/yemyoaung/managing-vehicle-tracking-tracking-svc/internal/config"
    "github.com/yemyoaung/managing-vehicle-tracking-tracking-svc/internal/handler"
    "github.com/yemyoaung/managing-vehicle-tracking-tracking-svc/internal/repositories"
    "github.com/yemyoaung/managing-vehicle-tracking-tracking-svc/internal/services"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var (
    ErrConfigMissing = errors.New("config is missing")
)

type App struct {
    validator  *validator.Validate
    cfg        *config.EnvConfig
    db         *mongo.Client
    rabbitConn *common.RabbitConnection
    shutdown   chan error
    exit       chan os.Signal
}

// NewApp creates a new App instance
func NewApp() *App {
    exit := make(chan os.Signal, 1)
    shutdown := make(chan error, 1)

    signal.Notify(exit, os.Interrupt, syscall.SIGTERM) // listen for termination signals

    go func() {
        defer close(exit)
        <-exit
        shutdown <- nil // shutdown
    }()

    return &App{shutdown: shutdown}
}

// SetValidator sets the validator for the app
func (a *App) SetValidator(validator *validator.Validate) *App {
    a.validator = validator
    return a
}

// SetConfig sets the environment configuration for the app
func (a *App) SetConfig(cfg *config.EnvConfig) *App {
    a.cfg = cfg
    return a
}

// Consume processes incoming tracking data messages from RabbitMQ
func (a *App) Consume(
    channel *amqp.Channel,
    trackingDataMessages <-chan amqp.Delivery,
    trackingService services.TrackingService,
) {
    for msg := range trackingDataMessages {
        go func(msg amqp.Delivery, channel *amqp.Channel) {
            var trackingData models.TrackingDataRequest
            if err := json.Unmarshal(msg.Body, &trackingData); err != nil {
                log.Printf("Failed to unmarshal message: %v", err)
                // Nack the message on error
                err := msg.Nack(false, false)
                if err != nil {
                    log.Println("Failed to nack message: ", err)
                    return
                }
                return
            }

            log.Println("Received tracking data: ", trackingData)

            // Track the vehicle using the service
            if err := trackingService.TrackVehicle(context.Background(), &trackingData); err != nil {
                log.Println("Failed to track vehicle: ", err)
                err := msg.Nack(false, false)
                if err != nil {
                    log.Println("Failed to nack message: ", err)
                    return
                }
                return
            }

            // Publish the result to a vehicle queue, for further processing 
            go func(body []byte) {
                err := channel.PublishWithContext(
                    context.Background(),
                    "",
                    a.cfg.VehicleQueue,
                    false,
                    false,
                    amqp.Publishing{
                        ContentType: common.ApplicationJSON,
                        Body:        body,
                    },
                )
                if err != nil {
                    log.Println("Failed to publish message: ", err)
                }
            }(msg.Body)

            // Acknowledge the message after processing
            if err := msg.Ack(false); err != nil {
                log.Println("Failed to ack message: ", err)
            }
        }(msg, channel)
    }
}

// Run starts the app, connects to MongoDB, RabbitMQ and consumes tracking data messages
func (a *App) Run(ctx context.Context) {
    var err error
    if a.cfg == nil {
        a.shutdown <- ErrConfigMissing
        return
    }

    // Connect to MongoDB
    a.db, err = mongo.Connect(ctx, options.Client().ApplyURI(a.cfg.DatabaseURL))
    if err != nil {
        a.shutdown <- err
        return
    }

    // Connect to RabbitMQ
    a.rabbitConn = common.NewRabbitConnection(a.cfg.RabbitmqUrl)
    channel, err := a.rabbitConn.Channel()
    if err != nil {
        a.shutdown <- err
        return
    }

    // Declare the tracking queue with durable
    _, err = channel.QueueDeclare(
        a.cfg.TrackingQueue,
        true,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        a.shutdown <- err
        return
    }

    // Start consuming messages from the declared queue
    trackingDataMessages, err := channel.Consume(
        a.cfg.TrackingQueue,
        "",
        false,
        false,
        false,
        false,
        nil,
    )
    if err != nil {
        a.shutdown <- err
        return
    }

    // Initialize the tracking service
    trackingRepo := repositories.NewMongoTackingRepository(a.db.Database("tracking"))
    trackingService := services.NewMongoTrackingService(trackingRepo)
    trackingHandler := handler.NewV1TrackingHandler(trackingService, a.validator)

    go a.Consume(channel, trackingDataMessages, trackingService)

    // Set up the HTTP server
    server := http.NewServeMux()

    // Set up the API routes
    v1Router := http.NewServeMux()                                                 // API version 1 router
    v1Router.HandleFunc("/api/v1/tracking-data", trackingHandler.FindTrackingData) // Vehicle creation and find

    // Apply middlewares and handle requests
    // The v1Router (which holds our API routes) will have two middlewares applied:
    // - CorsMiddleware: Adds CORS headers to the response
    // - LoggingMiddleware: Logs each incoming request for debugging and monitoring
    // - AuthorizationMiddleware: Authorizes the request using the auth service
    // - VerifySignatureMiddleware: Verifies the request's signature (ensuring it's from a trusted source)
    server.Handle(
        "/",
        common.CorsMiddleware(nil)(
            common.LoggingMiddleware(log.Default())(
                common.AuthorizationMiddleware[models.AuthUser](a.cfg.AuthSvc, a.cfg.SignatureKey)(
                    common.VerifySignatureMiddleware(a.cfg.SignatureKey)(
                        v1Router,
                    ),
                ),
            ),
        ),
    )

    log.Println("Vehicle service started on Port: ", a.cfg.Port)

    // Start the HTTP server in a goroutine
    go func() {
        err = http.ListenAndServe(a.cfg.Host+":"+a.cfg.Port, server)
        if !errors.Is(err, http.ErrServerClosed) {
            a.shutdown <- err
        }
    }()
}

// Shutdown gracefully shuts down the app
func (a *App) Shutdown(ctx context.Context) error {
    defer close(a.shutdown)

    // Disconnect from MongoDB
    defer func(ctx context.Context, db *mongo.Client) {
        if db == nil {
            return
        }
        err := db.Disconnect(ctx)
        if err != nil {
            log.Println("Failed to disconnect from database", err)
        }
    }(ctx, a.db)

    // Close RabbitMQ connection
    defer func(conn *common.RabbitConnection) {
        if conn == nil {
            return
        }
        err := conn.Close()
        if err != nil {
            log.Println("Failed to close rabbitmq connection", err)
        }
    }(a.rabbitConn)

    return <-a.shutdown
}
