package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"bank-service-cbr/config"
	appdb "bank-service-cbr/db"
	"bank-service-cbr/internal/handler"
	"bank-service-cbr/internal/middleware"
	"bank-service-cbr/internal/repository"
	"bank-service-cbr/internal/service"
)

func main() {
	log := logrus.New()
	log.SetFormatter(&logrus.JSONFormatter{})

	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("failed to load config")
	}

	level, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	db, err := appdb.NewDB(cfg)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to database")
	}
	defer db.Close()

	userRepository := repository.NewUserRepository(db)
	accountRepository := repository.NewAccountRepository(db)
	transactionRepository := repository.NewTransactionRepository(db)
	cardRepository := repository.NewCardRepository(db)
	creditRepository := repository.NewCreditRepository(db)
	scheduleRepository := repository.NewScheduleRepository(db)

	authService := service.NewAuthService(userRepository, cfg.JWTSecret, log)
	emailService := service.NewEmailService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, log)
	cbrService := service.NewCBRService(log)
	accountService := service.NewAccountService(db, accountRepository, transactionRepository, userRepository, emailService, log)
	transferService := service.NewTransferService(db, accountRepository, transactionRepository, userRepository, emailService, log)
	cardService := service.NewCardService(cardRepository, accountRepository, cfg.PGPPublicKey, cfg.PGPPrivateKey, []byte(cfg.HMACSecret), log)
	creditService := service.NewCreditService(db, creditRepository, scheduleRepository, accountRepository, transactionRepository, userRepository, cbrService, emailService, log)
	analyticsService := service.NewAnalyticsService(accountRepository, transactionRepository, creditRepository, scheduleRepository, log)

	schedulerService := service.NewSchedulerService(db, creditRepository, scheduleRepository, accountRepository, transactionRepository, userRepository, emailService, log)

	authHandler := handler.NewAuthHandler(authService)
	accountHandler := handler.NewAccountHandler(accountService)
	cardHandler := handler.NewCardHandler(cardService)
	transferHandler := handler.NewTransferHandler(transferService)
	creditHandler := handler.NewCreditHandler(creditService)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)

	router := mux.NewRouter()
	router.Use(middleware.LoggingMiddleware(log))

	router.HandleFunc("/register", authHandler.Register).Methods(http.MethodPost)
	router.HandleFunc("/login", authHandler.Login).Methods(http.MethodPost)

	api := router.PathPrefix("/").Subrouter()
	api.Use(middleware.AuthMiddleware(cfg.JWTSecret))

	api.HandleFunc("/accounts", accountHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/accounts", accountHandler.GetAll).Methods(http.MethodGet)

	api.HandleFunc("/accounts/{accountId}", accountHandler.GetOne).Methods(http.MethodGet)

	api.HandleFunc("/accounts/{accountId}/deposit", accountHandler.Deposit).Methods(http.MethodPost)
	api.HandleFunc("/accounts/{accountId}/withdraw", accountHandler.Withdraw).Methods(http.MethodPost)
	api.HandleFunc("/accounts/{accountId}/predict", analyticsHandler.GetPrediction).Methods(http.MethodGet)

	api.HandleFunc("/cards", cardHandler.Issue).Methods(http.MethodPost)
	api.HandleFunc("/cards", cardHandler.GetAll).Methods(http.MethodGet)

	api.HandleFunc("/transfer", transferHandler.Transfer).Methods(http.MethodPost)

	api.HandleFunc("/credits", creditHandler.TakeCredit).Methods(http.MethodPost)

	api.HandleFunc("/credits/{creditId}/schedule", creditHandler.GetSchedule).Methods(http.MethodGet)

	api.HandleFunc("/analytics", analyticsHandler.GetAnalytics).Methods(http.MethodGet)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go schedulerService.Start(ctx)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.WithField("addr", srv.Addr).Info("server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Error("server shutdown error")
	}
	log.Info("server stopped")
}
