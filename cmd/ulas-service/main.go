package main

import (
	"context"
	"log/slog"
	_ "net/http/pprof"
	"os"
	"time"

	"ulas-service/internal/aap"
	"ulas-service/internal/config"
	"ulas-service/internal/database"
	"ulas-service/internal/handler"
	"ulas-service/internal/ldap"
	"ulas-service/internal/repository"
	"ulas-service/internal/router"
	"ulas-service/internal/service"
	"ulas-service/internal/servicenow"
	"ulas-service/internal/token"
)

func main() {

	logHandler := config.LoggerInit()
	defer logHandler.Close()

	config.Load()
	cfg := config.Get()

	if err := database.Initialize(cfg); err != nil {
		slog.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	slog.Info("DB initialized successfully")
	db := database.SQLDB()
	defer db.Close()

	hostRepo := repository.NewPgHostRepository(db)
	hostService := service.NewDefaultHostService(hostRepo)
	hostHandler := handler.NewHostHandler(hostService)

	baselineRepo := repository.NewPgBaselineRepository(db)
	baselineService := service.NewDefaultBaselineService(baselineRepo, cfg)
	baselineHandler := handler.NewBaselineHandler(baselineService)

	deviationRepo := repository.NewPgDeviationRepository(db)
	deviationService := service.NewDefaultDeviationService(deviationRepo)
	deviationHandler := handler.NewDeviationHandler(deviationService)

	scanRepo := repository.NewPgScanRepository(db)
	snapshotRepo := repository.NewPgSnapshotRepository(db)
	incidentRepo := repository.NewPgIncidentRepository(db)
	snowTicketRepo := repository.NewPgServiceNowTicketRepository(db)
	scanScheduleRepo := repository.NewPgScanScheduleRepository(db)

	comparisonService := service.NewDefaultComparisonService(
		scanRepo,
		snapshotRepo,
		hostRepo,
		baselineRepo,
		deviationRepo,
		incidentRepo,
	)

	aapClient := aap.NewClient(cfg.AAPURL+cfg.AAPRESTVERSION, cfg.AAPUsername, cfg.AAPPassword)
	var aapSolarisClient *aap.Client
	if cfg.AAPSolarisURL != "" {
		aapSolarisClient = aap.NewClient(cfg.AAPSolarisURL+cfg.AAPRESTVERSIONSolaris, cfg.AAPUsernameSolaris, cfg.AAPPasswordSolaris)
	}
	scanService := service.NewDefaultScanService(
		scanRepo,
		hostRepo,
		snapshotRepo,
		incidentRepo,
		baselineRepo,
		aapClient,
		aapSolarisClient,
		comparisonService,
		cfg.AAPJobTemplateName,
		cfg.AAPJobTemplateNameSolaris,
		cfg.BackEndBaseUrl,
		cfg.AppStage,
		cfg,
	)
	scanHandler := handler.NewScanHandler(scanService)

	scanScheduleService := service.NewDefaultScanScheduleService(scanScheduleRepo)
	scanScheduleHandler := handler.NewScanScheduleHandler(scanScheduleService)

	scanScheduler := service.NewScanScheduler(scanScheduleRepo, scanService)
	scanScheduler.Start()
	defer scanScheduler.Stop()

	reportService := service.NewReportService(scanService)
	reportHandler := handler.NewReportHandler(reportService)

	incidentService := service.NewDefaultIncidentService(incidentRepo)
	snowClient := servicenow.NewClient(cfg.SNOWBaseURL, cfg.SNOWUsername, cfg.SNOWPassword)
	snowService := service.NewDefaultServiceNowService(incidentRepo, snowTicketRepo, scanRepo, hostRepo, snowClient)
	incidentHandler := handler.NewIncidentHandler(incidentService, snowService)

	snapshotService := service.NewDefaultSnapshotService(snapshotRepo)
	snapshotHandler := handler.NewSnapshotHandler(snapshotService)

	tokenMaker, err := token.NewJWTMaker(cfg.JWTSecretKey)
	if err != nil {
		slog.Error("failed to create token maker", "error", err)
		os.Exit(1)
	}

	var ldapClient *ldap.Client
	if cfg.LDAPServer != "" {
		ldapClient = &ldap.Client{
			Host:               cfg.LDAPServer,
			Port:               cfg.LDAPPort,
			Base:               cfg.LDAPBaseDN,
			BindDN:             cfg.ResolvedLDAPBindDN(),
			BindPassword:       cfg.LDAPBindPassword,
			UserFilter:         cfg.LDAPUserFilter,
			GroupFilter:        cfg.LDAPGroupFilter,
			Attributes:         []string{"givenName", "sn", "mail", "memberOf", "cn"},
			UseSSL:             cfg.LDAPUseSSL,
			SkipTLS:            cfg.LDAPSkipTLS,
			InsecureSkipVerify: true,
		}
	}

	authHandler := handler.NewAuthHandler(tokenMaker, ldapClient, cfg)

	healthHandler := handler.NewHealthHandler(db, aapClient, aapSolarisClient)
	r := router.New(
		authHandler,
		tokenMaker,
		healthHandler,
		hostHandler,
		baselineHandler,
		deviationHandler,
		scanHandler,
		reportHandler,
		incidentHandler,
		snapshotHandler,
		scanScheduleHandler,
	)

	//AAP job status poller.
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := scanService.PollActiveScans(context.Background()); err != nil {
					slog.Error("scan poll failed", "error", err)
				}
			}
		}
	}()

	addr := ":" + cfg.Port
	slog.Info("starting ulas-service", "addr", addr, "stage", cfg.AppStage)
	if err := r.Run(addr); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
