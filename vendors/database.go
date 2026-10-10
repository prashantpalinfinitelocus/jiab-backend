package vendors

import (
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/elliotchance/sshtunnel"
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
)

var globalTunnel *sshtunnel.SSHTunnel
var tunnelMutex sync.Mutex

// checkPortAvailable checks if a port is available for binding
func checkPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// CloseTunnel closes the global SSH tunnel if it exists
func CloseTunnel() {
	tunnelMutex.Lock()
	defer tunnelMutex.Unlock()
	if globalTunnel != nil {
		globalTunnel.Close()
		globalTunnel = nil
		log.Info("SSH tunnel closed")
	}
}

// InitDatabase connects to the configured database, or terminates the
// process (log.Fatalf) if it can't. Use this in binaries where the database
// is a hard dependency and failing fast on startup is the right behavior
// (e.g. the main API server).
func InitDatabase() *gorm.DB {
	db, err := connectDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	return db
}

// TryInitDatabase connects to the configured database like InitDatabase, but
// returns an error instead of terminating the process on failure. Use this
// in binaries where the database is an optional/secondary dependency (e.g.
// the Pub/Sub subscriber, where DB access is only needed to invalidate dead
// FCM tokens — its core job of relaying push notifications must keep working
// even if the DB is unreachable or unconfigured).
func TryInitDatabase() (*gorm.DB, error) {
	return connectDatabase()
}

func connectDatabase() (*gorm.DB, error) {
	cfg := config.GetConfig()
	if cfg.SSHConfig.UseSSH == "true" {
		if err := ensureSSHTunnel(cfg); err != nil {
			return nil, err
		}
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.DbConfig.Host,
		cfg.DbConfig.Port,
		cfg.DbConfig.User,
		cfg.DbConfig.Password,
		cfg.DbConfig.DBName,
		cfg.DbConfig.SSLMode,
	)

	gormLogger := logger.Default.LogMode(logger.Info)
	if cfg.AppEnv == "production" {
		gormLogger = logger.Default.LogMode(logger.Error)
	}

	db, err := gorm.Open(postgres.New(postgres.Config{
		DriverName: "pgx",
		DSN:        dsn,
	}), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	log.Info("Database connected successfully")

	return db, nil
}

// ensureSSHTunnel starts (or reuses) the global SSH tunnel used to reach the
// database when USE_SSH=true. Returns an error instead of exiting the
// process on failure, so callers can decide whether to fail fast or degrade.
func ensureSSHTunnel(cfg *config.Config) error {
	tunnelMutex.Lock()
	if globalTunnel != nil {
		tunnelMutex.Unlock()
		log.Info("Reusing existing SSH tunnel")
		return nil
	}

	if _, err := os.Stat("/app/ssh-key.pem"); err != nil {
		tunnelMutex.Unlock()
		if os.IsNotExist(err) {
			return fmt.Errorf("SSH key file not found at /app/ssh-key.pem")
		}
		return fmt.Errorf("error accessing SSH key file: %w", err)
	}

	// Check if port is available
	if !checkPortAvailable(cfg.DbConfig.Port) {
		log.Warnf("Port %d is already in use, waiting for it to be available...", cfg.DbConfig.Port)
		// Wait a bit and try again
		time.Sleep(5 * time.Second)
		if !checkPortAvailable(cfg.DbConfig.Port) {
			tunnelMutex.Unlock()
			return fmt.Errorf("port %d is still in use after waiting; check if another tunnel is running", cfg.DbConfig.Port)
		}
	}

	tunnel, err := sshtunnel.NewSSHTunnel(
		fmt.Sprintf("%s@%s",
			cfg.SSHConfig.User,
			cfg.SSHConfig.Host,
		),
		sshtunnel.PrivateKeyFile("/app/ssh-key.pem"),
		fmt.Sprintf("%s:%d",
			cfg.SSHConfig.RDSEndpoint,
			cfg.SSHConfig.RDSPort,
		),
		fmt.Sprintf("%d", cfg.DbConfig.Port),
	)
	if err != nil {
		tunnelMutex.Unlock()
		return fmt.Errorf("failed to create SSH tunnel: %w", err)
	}

	// Store tunnel globally for cleanup
	globalTunnel = tunnel

	// Add error handling for tunnel start
	go func() {
		if err := tunnel.Start(); err != nil {
			log.Errorf("SSH tunnel error: %v", err)
			tunnelMutex.Lock()
			defer tunnelMutex.Unlock()
			globalTunnel = nil
		}
	}()
	tunnelMutex.Unlock()

	time.Sleep(3 * time.Second)
	log.Info("SSH tunnel started")
	return nil
}
