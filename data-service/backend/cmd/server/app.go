package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	kratos "github.com/go-kratos/kratos/v3"
	"github.com/go-kratos/kratos/v3/transport"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	entitybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity"
	chainnodebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/chainnode"
	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
	conceptbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/concept"
	countrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/country"
	industrybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industry"
	industrychainbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/industrychain"
	organizationbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/organization"
	eventbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/event"
	evidencebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/evidence"
	reportbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/report"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/runtimehealth"
	sourcebiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/source"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/conf"
	data "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/dbmigration"
	entitydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity"
	chainnodedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/chainnode"
	companydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/company"
	conceptdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/concept"
	countrydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/country"
	industrydata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/industry"
	industrychaindata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/industrychain"
	organizationdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/organization"
	eventdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/event"
	evidencedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/evidence"
	reportdata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/report"
	sourcedata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/source"
	"github.com/meierlink88/tidewise-ai/data-service/backend/internal/server"
	chainnodeservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/chainnode"
	companyservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/company"
	conceptservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/concept"
	countryservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/country"
	industryservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/industry"
	industrychainservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/industrychain"
	organizationservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/entity/organization"
	eventservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/event"
	evidenceservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/evidence"
	reportservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/report"
	researchservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/research"
	runtimehealthservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/runtimehealth"
	sourceservice "github.com/meierlink88/tidewise-ai/data-service/backend/internal/service/source"
)

const applicationStopTimeout = 10 * time.Second
const resourceCleanupTimeout = 5 * time.Second

func buildApp(config conf.Config, logger *slog.Logger) (*kratos.App, func(context.Context) error, error) {
	authenticator, err := buildAuthenticator(config)
	if err != nil {
		return nil, nil, err
	}

	connectTimeout := time.Duration(config.Database.ConnectTimeoutSeconds) * time.Second
	databaseContext, cancelDatabase := context.WithTimeout(context.Background(), connectTimeout)
	db, err := data.OpenPostgres(databaseContext, config)
	cancelDatabase()
	if err != nil {
		return nil, nil, fmt.Errorf("open Data PostgreSQL: %w", err)
	}

	readinessContext, cancelReadiness := context.WithTimeout(context.Background(), connectTimeout)
	_, err = dbmigration.RequirePostgresReadyReadOnly(readinessContext, db, config.Migration.Directory)
	cancelReadiness()
	if err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("check read-only migration readiness: %w", err)
	}

	closeBuildResources := func(buildErr error) error {
		return errors.Join(buildErr, db.Close())
	}
	evidenceStore, err := evidencedata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Evidence store: %w", err))
	}
	evidenceUseCase, err := evidencebiz.NewUseCase(evidenceStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Evidence use case: %w", err))
	}
	eventStore, err := eventdata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event store: %w", err))
	}
	eventUseCase, err := eventbiz.NewUseCase(eventStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event use case: %w", err))
	}
	entityStore, err := entitydata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Entity store: %w", err))
	}
	entityUseCase, err := entitybiz.NewUseCase(entityStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Entity use case: %w", err))
	}
	countryStore, err := countrydata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Country store: %w", err))
	}
	countryUseCase, err := countrybiz.NewUseCase(countryStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Country use case: %w", err))
	}
	industryStore, err := industrydata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Industry store: %w", err))
	}
	industryUseCase, err := industrybiz.NewUseCase(industryStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Industry use case: %w", err))
	}
	conceptStore, err := conceptdata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Concept store: %w", err))
	}
	conceptUseCase, err := conceptbiz.NewUseCase(conceptStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Concept use case: %w", err))
	}
	chainNodeStore, err := chainnodedata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure ChainNode store: %w", err))
	}
	chainNodeUseCase, err := chainnodebiz.NewUseCase(chainNodeStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure ChainNode use case: %w", err))
	}
	industryChainStore, err := industrychaindata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure IndustryChain store: %w", err))
	}
	industryChainUseCase, err := industrychainbiz.NewUseCase(industryChainStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure IndustryChain use case: %w", err))
	}
	organizationStore, err := organizationdata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Organization store: %w", err))
	}
	organizationUseCase, err := organizationbiz.NewUseCase(organizationStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Organization use case: %w", err))
	}
	sourceStore, err := sourcedata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Source store: %w", err))
	}
	sourceUseCase, err := sourcebiz.NewUseCase(sourceStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Source use case: %w", err))
	}
	companyStore, err := companydata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Company store: %w", err))
	}
	companyUseCase, err := companybiz.NewProjectionUseCase(companyStore)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Company projection use case: %w", err))
	}
	researchUseCase, err := research.NewUseCase(entityUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Research use case: %w", err))
	}
	reportStore, err := reportdata.NewStore(db)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Report store: %w", err))
	}
	reportUseCase, err := reportbiz.NewUseCase(reportStore, time.Now)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Report use case: %w", err))
	}

	runtimeHealthApplication, err := runtimehealthservice.NewService(runtimehealth.New(time.Now))
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Runtime Health API service: %w", err))
	}
	researchApplication, err := researchservice.NewService(researchUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Research API service: %w", err))
	}
	evidenceApplication, err := evidenceservice.NewService(evidenceUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Evidence API service: %w", err))
	}
	eventApplication, err := eventservice.NewService(eventUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Event API service: %w", err))
	}
	countryApplication, err := countryservice.NewService(countryUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Country API service: %w", err))
	}
	industryApplication, err := industryservice.NewService(industryUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Industry API service: %w", err))
	}
	conceptApplication, err := conceptservice.NewService(conceptUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Concept API service: %w", err))
	}
	chainNodeApplication, err := chainnodeservice.NewService(chainNodeUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure ChainNode API service: %w", err))
	}
	industryChainApplication, err := industrychainservice.NewService(industryChainUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure IndustryChain API service: %w", err))
	}
	organizationApplication, err := organizationservice.NewService(organizationUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Organization API service: %w", err))
	}
	sourceApplication, err := sourceservice.NewService(sourceUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Source API service: %w", err))
	}
	companyApplication, err := companyservice.NewService(companyUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Company projection API service: %w", err))
	}
	reportApplication, err := reportservice.NewService(reportUseCase)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure Report API service: %w", err))
	}
	httpServer, err := server.NewHTTPServer(config, runtimeHealthApplication, researchApplication, eventApplication, evidenceApplication, countryApplication, industryApplication, conceptApplication, chainNodeApplication, industryChainApplication, organizationApplication, sourceApplication, companyApplication, reportApplication, authenticator, logger)
	if err != nil {
		return nil, nil, closeBuildResources(fmt.Errorf("configure HTTP server: %w", err))
	}

	return newApp(httpServer, logger), func(ctx context.Context) error {
		return db.Close()
	}, nil
}

func buildAuthenticator(config conf.Config) (*server.Authenticator, error) {
	credentials := []server.Credential{
		{
			Secret: config.Secrets.ServiceToken,
			Principal: v1.Principal{Identity: "tidewise-internal-service", Scopes: []string{
				server.ScopeRawEvidenceImport,
				server.ScopeRawEvidenceRead,
				server.ScopeEvidenceImport,
				server.ScopeEvidenceCategoryRead,
				server.ScopeResearchRead,
				server.ScopeAdminRead,
				server.ScopeEventPublish,
				server.ScopeCountryRead,
				server.ScopeCountryWrite,
				server.ScopeIndustryRead,
				server.ScopeIndustryWrite,
				server.ScopeConceptRead,
				server.ScopeConceptWrite,
				server.ScopeChainNodeRead,
				server.ScopeChainNodeWrite,
				server.ScopeIndustryChainRead,
				server.ScopeIndustryChainWrite,
				server.ScopeOrganizationRead,
				server.ScopeOrganizationWrite,
				server.ScopeSourceRead,
				server.ScopeSourceWrite,
				server.ScopeCompanyRead,
				server.ScopeReportRead,
				server.ScopeReportPublish,
			}},
		},
	}
	authenticator, err := server.NewAuthenticator(credentials)
	if err != nil {
		return nil, fmt.Errorf("build Data authenticator: %w", err)
	}
	return authenticator, nil
}

func newApp(httpServer transport.Server, logger *slog.Logger) *kratos.App {
	return kratos.New(
		kratos.Name(conf.ServiceName),
		kratos.Version(conf.ServiceVersion),
		kratos.Logger(logger),
		kratos.StopTimeout(applicationStopTimeout),
		kratos.Server(httpServer),
	)
}

func runApplication(app interface{ Run() error }, cleanup func(context.Context) error) error {
	runErr := app.Run()
	cleanupErr := runCleanup(cleanup, resourceCleanupTimeout)
	return errors.Join(runErr, cleanupErr)
}

func runCleanup(cleanup func(context.Context) error, timeout time.Duration) error {
	if cleanup == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- cleanup(ctx)
	}()

	select {
	case err := <-result:
		if err != nil {
			return fmt.Errorf("cleanup Data resources: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("cleanup Data resources: %w", ctx.Err())
	}
}
