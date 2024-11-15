package main

import (
    "context"
    "log"

    "github.com/go-playground/validator/v10"
    "github.com/yemyoaung/managing-vehicle-tracking-common"
    "github.com/yemyoaung/managing-vehicle-tracking-tracking-svc/internal/app"
    "github.com/yemyoaung/managing-vehicle-tracking-tracking-svc/internal/config"
)

func main() {
    validate := validator.New(
        validator.WithRequiredStructEnabled(),
    )
    load, err := common.NewConfigLoaderFromEnvFile[config.EnvConfig](".env", validate)
    if err != nil {
        log.Fatal("Failed to load config")
    }

    ctx := context.Background()

    instance := app.NewApp().SetValidator(validate).SetConfig(load.Config)

    instance.Run(ctx)

    err = instance.Shutdown(ctx)

    if err != nil {
        log.Fatal("Shutdown failed:", err)
        return
    }
    log.Println("App shutdown successfully")
}
