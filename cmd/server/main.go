package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"

	"github.com/loks1k192/task-manager/internal/circuitbreaker"
	"github.com/loks1k192/task-manager/internal/config"
	delivery "github.com/loks1k192/task-manager/internal/delivery/http"
	mysqlrepo "github.com/loks1k192/task-manager/internal/repository/mysql"
	redisrepo "github.com/loks1k192/task-manager/internal/repository/redis"
	"github.com/loks1k192/task-manager/internal/usecase"
)

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := sqlx.Connect("mysql", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	if err := runMigrations(db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	userRepo := mysqlrepo.NewUserRepository(db)
	teamRepo := mysqlrepo.NewTeamRepository(db)
	taskRepo := mysqlrepo.NewTaskRepository(db)
	commentRepo := mysqlrepo.NewCommentRepository(db)
	cacheRepo := redisrepo.NewCacheRepository(rdb)

	emailSvc := circuitbreaker.NewEmailService()

	authUC := usecase.NewAuthUseCase(userRepo, cfg.JWT.Secret, cfg.JWT.Expiration)
	teamUC := usecase.NewTeamUseCase(teamRepo, userRepo, emailSvc)
	taskUC := usecase.NewTaskUseCase(taskRepo, teamRepo, commentRepo, cacheRepo)

	router := delivery.NewRouter(authUC, teamUC, taskUC)

	srv := &http.Server{
		Addr:    cfg.Server.Port,
		Handler: router,
	}

	go func() {
		log.Printf("server starting on %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server stopped gracefully")
}

func runMigrations(db *sqlx.DB) error {
	migration, err := os.ReadFile("migrations/001_schema.up.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(migration))
	return err
}
