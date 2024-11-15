package config

type EnvConfig struct {
    Host          string `json:"HOST" validate:"required"`
    Port          string `json:"PORT" validate:"required"`
    DatabaseURL   string `json:"DATABASE_URL" validate:"required"`
    RabbitmqUrl   string `json:"RABBITMQ_URL" validate:"required"`
    TrackingQueue string `json:"TRACKING_QUEUE" validate:"required"`
    VehicleQueue  string `json:"VEHICLE_QUEUE" validate:"required"`
    SignatureKey  string `json:"SIGNATURE_KEY" validate:"required"`
    AuthSvc       string `json:"AUTH_SVC" validate:"required"`
}
