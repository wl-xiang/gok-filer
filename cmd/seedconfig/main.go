// Command seedconfig provisions a ready-to-use Gokapi configuration without the
// interactive web setup wizard. Used for local / docker-compose deployments.
//
// TEMPORARY TOOL - delete this directory after the deployment has been seeded.
//
// Environment variables (all optional, sensible defaults for docker compose):
//
//	GOKAPI_CONFIG_DIR   -> directory the config.json is written to
//	GOKAPI_DATA_DIR     -> data directory recorded in the configuration
//	SEED_DATABASE_URL   -> sqlite URL used to create the database
//	SEED_ADMIN_NAME / SEED_ADMIN_PASSWORD -> the super admin account
//	SEED_SERVER_URL / SEED_PUBLIC_NAME   -> public URL and display name
package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/forceu/gokapi/internal/configuration"
	"github.com/forceu/gokapi/internal/configuration/configupgrade"
	"github.com/forceu/gokapi/internal/encryption"
	"github.com/forceu/gokapi/internal/environment"
	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/models"
)

func main() {
	env := environment.New()

	superAdminName := getEnv("SEED_ADMIN_NAME", "root")
	superAdminPassword := getEnv("SEED_ADMIN_PASSWORD", "root1234")

	config := models.Configuration{
		Authentication: models.AuthenticationConfig{
			Method:    models.AuthenticationInternal,
			Username:  superAdminName,
			SaltAdmin: helper.GenerateRandomString(30),
			SaltFiles: helper.GenerateRandomString(30),
		},
		Port:               ":" + strconv.Itoa(env.WebserverPort),
		ServerUrl:          getEnv("SEED_SERVER_URL", "http://localhost:53842/"),
		RedirectUrl:        "/index",
		PublicName:         getEnv("SEED_PUBLIC_NAME", "Gokapi"),
		DataDir:            env.DataDir,
		DatabaseUrl:        getEnv("SEED_DATABASE_URL", "sqlite://gokapi-data/gokapi.sqlite"),
		ConfigVersion:      configupgrade.CurrentConfigVersion,
		MaxFileSizeMB:      env.MaxFileSize,
		MaxMemory:          env.MaxMemory,
		ChunkSize:          env.ChunkSizeMB,
		MaxParallelUploads: env.MaxParallelUploads,
		Encryption:         models.Encryption{Level: encryption.NoEncryption},
		UseSsl:             false,
		SaveIp:             true,
		IncludeFilename:    true,
	}

	// Writes config.json, creates the database and creates the super admin user
	configuration.LoadFromSetup(config, nil, configuration.End2EndReconfigParameters{},
		configuration.HashPassword(superAdminPassword, false, ""))

	configPath, _, _, awsConfigPath := environment.GetConfigPaths()
	fmt.Println("Configuration written to: " + configPath)
	fmt.Println("Cloud config path:        " + awsConfigPath)
	fmt.Println("Data directory:           " + config.DataDir)
	fmt.Println("Database:                 " + config.DatabaseUrl)
	fmt.Println("Super admin:              " + superAdminName + " / " + superAdminPassword)
	fmt.Println("Seeding finished.")
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
