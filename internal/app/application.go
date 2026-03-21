package app

import (
	"fmt"

	"github.com/konstfish/pumice/internal/assets"
	"github.com/konstfish/pumice/internal/collector"
	"github.com/konstfish/pumice/internal/config"
	"github.com/konstfish/pumice/internal/processor"
	"github.com/konstfish/pumice/internal/renderer"
	"github.com/konstfish/pumice/internal/resolver"
	"github.com/konstfish/pumice/internal/server"
	"github.com/konstfish/pumice/internal/sitebuilder"
)

type Application struct {
	configManager     *config.Manager
	fileCollector     *collector.FileCollector
	linkResolver      *resolver.LinkResolver
	assetManager      *assets.Manager
	markdownProcessor *processor.MarkdownProcessor
	pageRenderer      *renderer.PageRenderer
	siteBuilder       *sitebuilder.SiteBuilder
	webServer         *server.Server
}

func NewApplication() *Application {
	app := &Application{}
	app.initializeComponents()
	return app
}

func (a *Application) initializeComponents() {
	a.configManager = config.NewManager()
}

func (a *Application) LoadConfig(args []string) error {
	return a.configManager.Load(args)
}

func (a *Application) ShouldServe() bool {
	return a.configManager.ShouldServe()
}

func (a *Application) GetPort() string {
	return a.configManager.GetPort()
}

func (a *Application) initializeBuildComponents() {
	contentDir := a.configManager.GetContentDir()
	buildDir := a.configManager.GetBuildDir()

	a.fileCollector = collector.NewFileCollector(contentDir)
	
	a.linkResolver = resolver.NewLinkResolver(contentDir, buildDir, a.fileCollector)
	
	a.assetManager = assets.NewManager(buildDir, a.configManager.GetBasePath(), a.fileCollector)
	
	a.markdownProcessor = processor.NewMarkdownProcessor(a.linkResolver)
	
	a.pageRenderer = renderer.NewPageRenderer(a.configManager, a.assetManager)
	
	a.siteBuilder = sitebuilder.NewSiteBuilder(
		contentDir,
		buildDir,
		a.configManager.GetStaticDir(),
		a.configManager.GetSiteURL(),
		a.configManager.GetPageTitle(),
		a.configManager.GetBasePath(),
		a.fileCollector,
		a.linkResolver,
		a.assetManager,
		a.markdownProcessor,
		a.pageRenderer,
	)
}

func (a *Application) Build() error {
	a.initializeBuildComponents()
	
	if err := a.siteBuilder.Build(); err != nil {
		return fmt.Errorf("building site: %w", err)
	}
	
	fmt.Println("Build completed successfully!")
	return nil
}

func (a *Application) Serve(port string) error {
	buildDir := a.configManager.GetBuildDir()
	a.webServer = server.New(buildDir, port)
	return a.webServer.Start()
}

func (a *Application) ServeWithLogging(port string) {
	buildDir := a.configManager.GetBuildDir()
	a.webServer = server.New(buildDir, port)
	a.webServer.StartWithLogging()
}