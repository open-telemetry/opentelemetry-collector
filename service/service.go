// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package service handles the command-line, configuration, and runs the
// OpenTelemetry Collector.
package service

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"runtime"
	"sort"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/config/configcheck"
	"go.opentelemetry.io/collector/config/configmodels"
	"go.opentelemetry.io/collector/config/configtelemetry"
	"go.opentelemetry.io/collector/internal/collector/telemetry"
	"go.opentelemetry.io/collector/internal/version"
	"go.opentelemetry.io/collector/service/builder"
	"go.opentelemetry.io/collector/service/internal"
)

const (
	servicezPath   = "servicez"
	pipelinezPath  = "pipelinez"
	extensionzPath = "extensionz"
)

// State defines Application's state.
type State int

const (
	Starting State = iota
	Running
	Closing
	Closed
)

// GetStateChannel returns state channel of the application.
func (app *Application) GetStateChannel() chan State {
	return app.stateChannel
}

// Application represents a collector application
type Application struct {
	info            component.ApplicationStartInfo
	rootCmd         *cobra.Command
	v               *viper.Viper
	logger          *zap.Logger
	builtExporters  builder.Exporters
	builtReceivers  builder.Receivers
	builtPipelines  builder.BuiltPipelines
	builtExtensions builder.Extensions
	stateChannel    chan State

	factories component.Factories
	config    *configmodels.Config

	// stopTestChan is used to terminate the application in end to end tests.
	stopTestChan chan struct{}

	// signalsChannel is used to receive termination signals from the OS.
	signalsChannel chan os.Signal

	// asyncErrorChannel is used to signal a fatal error from any component.
	asyncErrorChannel chan error
}

// Command returns Application's root command.
func (app *Application) Command() *cobra.Command {
	return app.rootCmd
}

// Parameters holds configuration for creating a new Application.
type Parameters struct {
	// Factories component factories.
	Factories component.Factories
	// ApplicationStartInfo provides application start information.
	ApplicationStartInfo component.ApplicationStartInfo
	// ConfigFactory that creates the configuration.
	// If it is not provided the default factory (FileLoaderConfigFactory) is used.
	// The default factory loads the configuration file and overrides component's configuration
	// properties supplied via --set command line flag.
	ConfigFactory ConfigFactory
	// LoggingOptions provides a way to change behavior of zap logging.
	LoggingOptions []zap.Option
}

// ConfigFactory creates config.
// The ConfigFactory implementation should call AddSetFlagProperties to enable configuration passed via `--set` flag.
// Viper and command instances are passed from the Application.
// The factories also belong to the Application and are equal to the factories passed via Parameters.
type ConfigFactory func(v *viper.Viper, cmd *cobra.Command, factories component.Factories) (*configmodels.Config, error)

// FileLoaderConfigFactory implements ConfigFactory and it creates configuration from file
// and from --set command line flag (if the flag is present).
func FileLoaderConfigFactory(v *viper.Viper, cmd *cobra.Command, factories component.Factories) (*configmodels.Config, error) {
	file := builder.GetConfigFile()
	if file == "" {
		return nil, errors.New("config file not specified")
	}
	// first load the config file
	v.SetConfigFile(file)
	err := v.ReadInConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config file %q: %v", file, err)
	}

	// next overlay the config file with --set flags
	if err := AddSetFlagProperties(v, cmd); err != nil {
		return nil, fmt.Errorf("failed to process set flag: %v", err)
	}
	return config.Load(v, factories)
}

// New creates and returns a new instance of Application.
func New(params Parameters) (*Application, error) {
	app := &Application{
		info:         params.ApplicationStartInfo,
		v:            config.NewViper(),
		factories:    params.Factories,
		stateChannel: make(chan State, Closed+1),
	}

	factory := params.ConfigFactory
	if factory == nil {
		// use default factory that loads the configuration file
		factory = FileLoaderConfigFactory
	}

	rootCmd := &cobra.Command{
		Use:  params.ApplicationStartInfo.ExeName,
		Long: params.ApplicationStartInfo.LongName,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := app.init(params.LoggingOptions)
			if err != nil {
				return err
			}

			err = app.execute(context.Background(), factory)
			if err != nil {
				return err
			}

			return nil
		},
	}

	// TODO: coalesce this code and expose this information to other components.
	flagSet := new(flag.FlagSet)
	addFlagsFns := []func(*flag.FlagSet){
		configtelemetry.Flags,
		telemetry.Flags,
		builder.Flags,
		loggerFlags,
	}
	for _, addFlags := range addFlagsFns {
		addFlags(flagSet)
	}
	rootCmd.Flags().AddGoFlagSet(flagSet)
	addSetFlag(rootCmd.Flags())

	app.rootCmd = rootCmd

	return app, nil
}

// ReportFatalError is used to report to the host that the receiver encountered
// a fatal error (i.e.: an error that the instance can't recover from) after
// its start function has already returned.
func (app *Application) ReportFatalError(err error) {
	app.asyncErrorChannel <- err
}

// GetLogger returns logger used by the Application.
// The logger is initialized after application start.
func (app *Application) GetLogger() *zap.Logger {
	return app.logger
}

func (app *Application) GetFactory(kind component.Kind, componentType configmodels.Type) component.Factory {
	switch kind {
	case component.KindReceiver:
		return app.factories.Receivers[componentType]
	case component.KindProcessor:
		return app.factories.Processors[componentType]
	case component.KindExporter:
		return app.factories.Exporters[componentType]
	case component.KindExtension:
		return app.factories.Extensions[componentType]
	}
	return nil
}

func (app *Application) GetExtensions() map[configmodels.Extension]component.ServiceExtension {
	return app.builtExtensions.ToMap()
}

func (app *Application) GetExporters() map[configmodels.DataType]map[configmodels.Exporter]component.Exporter {
	return app.builtExporters.ToMapByDataType()
}

func (app *Application) RegisterZPages(mux *http.ServeMux, pathPrefix string) {
	mux.HandleFunc(path.Join(pathPrefix, servicezPath), app.handleServicezRequest)
	mux.HandleFunc(path.Join(pathPrefix, pipelinezPath), app.handlePipelinezRequest)
	mux.HandleFunc(path.Join(pathPrefix, extensionzPath), app.handleExtensionzRequest)
}

func (app *Application) Shutdown() {
	// TODO: Implement a proper shutdown with graceful draining of the pipeline.
	// See https://github.com/open-telemetry/opentelemetry-collector/issues/483.
	defer func() {
		if r := recover(); r != nil {
			app.logger.Info("stopTestChan already closed")
		}
	}()
	close(app.stopTestChan)
}

func (app *Application) init(options []zap.Option) error {
	l, err := newLogger(options)
	if err != nil {
		return fmt.Errorf("failed to get logger: %w", err)
	}
	app.logger = l
	return nil
}

func (app *Application) setupTelemetry(ballastSizeBytes uint64) error {
	app.logger.Info("Setting up own telemetry...")

	err := applicationTelemetry.init(app.asyncErrorChannel, ballastSizeBytes, app.logger)
	if err != nil {
		return fmt.Errorf("failed to initialize telemetry: %w", err)
	}

	return nil
}

// runAndWaitForShutdownEvent waits for one of the shutdown events that can happen.
func (app *Application) runAndWaitForShutdownEvent() {
	app.logger.Info("Everything is ready. Begin running and processing data.")

	// plug SIGTERM signal into a channel.
	app.signalsChannel = make(chan os.Signal, 1)
	signal.Notify(app.signalsChannel, os.Interrupt, syscall.SIGTERM)

	// set the channel to stop testing.
	app.stopTestChan = make(chan struct{})
	app.stateChannel <- Running
	select {
	case err := <-app.asyncErrorChannel:
		app.logger.Error("Asynchronous error received, terminating process", zap.Error(err))
	case s := <-app.signalsChannel:
		app.logger.Info("Received signal from OS", zap.String("signal", s.String()))
	case <-app.stopTestChan:
		app.logger.Info("Received stop test request")
	}
	app.stateChannel <- Closing
}

func (app *Application) setupConfigurationComponents(ctx context.Context, factory ConfigFactory) error {
	if err := configcheck.ValidateConfigFromFactories(app.factories); err != nil {
		return err
	}

	app.logger.Info("Loading configuration...")
	cfg, err := factory(app.v, app.rootCmd, app.factories)
	if err != nil {
		return fmt.Errorf("cannot load configuration: %w", err)
	}
	err = config.ValidateConfig(cfg, app.logger)
	if err != nil {
		return fmt.Errorf("cannot load configuration: %w", err)
	}

	app.config = cfg
	app.logger.Info("Applying configuration...")

	err = app.setupExtensions(ctx)
	if err != nil {
		return fmt.Errorf("cannot setup extensions: %w", err)
	}

	err = app.setupPipelines(ctx)
	if err != nil {
		return fmt.Errorf("cannot setup pipelines: %w", err)
	}

	return nil
}

func (app *Application) setupExtensions(ctx context.Context) error {
	var err error
	app.builtExtensions, err = builder.NewExtensionsBuilder(app.logger, app.info, app.config, app.factories.Extensions).Build()
	if err != nil {
		return fmt.Errorf("cannot build builtExtensions: %w", err)
	}
	app.logger.Info("Starting extensions...")
	return app.builtExtensions.StartAll(ctx, app)
}

func (app *Application) setupPipelines(ctx context.Context) error {
	// Pipeline is built backwards, starting from exporters, so that we create objects
	// which are referenced before objects which reference them.

	// First create exporters.
	var err error
	app.builtExporters, err = builder.NewExportersBuilder(app.logger, app.info, app.config, app.factories.Exporters).Build()
	if err != nil {
		return fmt.Errorf("cannot build builtExporters: %w", err)
	}

	app.logger.Info("Starting exporters...")
	err = app.builtExporters.StartAll(ctx, app)
	if err != nil {
		return fmt.Errorf("cannot start builtExporters: %w", err)
	}

	// Create pipelines and their processors and plug exporters to the
	// end of the pipelines.
	app.builtPipelines, err = builder.NewPipelinesBuilder(app.logger, app.info, app.config, app.builtExporters, app.factories.Processors).Build()
	if err != nil {
		return fmt.Errorf("cannot build pipelines: %w", err)
	}

	app.logger.Info("Starting processors...")
	err = app.builtPipelines.StartProcessors(ctx, app)
	if err != nil {
		return fmt.Errorf("cannot start processors: %w", err)
	}

	// Create receivers and plug them into the start of the pipelines.
	app.builtReceivers, err = builder.NewReceiversBuilder(app.logger, app.info, app.config, app.builtPipelines, app.factories.Receivers).Build()
	if err != nil {
		return fmt.Errorf("cannot build receivers: %w", err)
	}

	app.logger.Info("Starting receivers...")
	err = app.builtReceivers.StartAll(ctx, app)
	if err != nil {
		return fmt.Errorf("cannot start receivers: %w", err)
	}

	return nil
}

func (app *Application) shutdownPipelines(ctx context.Context) error {
	// Shutdown order is the reverse of building: first receivers, then flushing pipelines
	// giving senders a chance to send all their data. This may take time, the allowed
	// time should be part of configuration.

	var errs []error

	app.logger.Info("Stopping receivers...")
	err := app.builtReceivers.ShutdownAll(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to stop receivers: %w", err))
	}

	app.logger.Info("Stopping processors...")
	err = app.builtPipelines.ShutdownProcessors(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown processors: %w", err))
	}

	app.logger.Info("Stopping exporters...")
	err = app.builtExporters.ShutdownAll(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown exporters: %w", err))
	}

	return componenterror.CombineErrors(errs)
}

func (app *Application) shutdownExtensions(ctx context.Context) error {
	app.logger.Info("Stopping extensions...")
	err := app.builtExtensions.ShutdownAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to shutdown extensions: %w", err)
	}
	return nil
}

func (app *Application) execute(ctx context.Context, factory ConfigFactory) error {
	app.logger.Info("Starting "+app.info.LongName+"...",
		zap.String("Version", app.info.Version),
		zap.String("GitHash", app.info.GitHash),
		zap.Int("NumCPU", runtime.NumCPU()),
	)
	app.stateChannel <- Starting

	// Set memory ballast
	ballast, ballastSizeBytes := app.createMemoryBallast()

	app.asyncErrorChannel = make(chan error)

	// Setup everything.
	err := app.setupTelemetry(ballastSizeBytes)
	if err != nil {
		return err
	}

	err = app.setupConfigurationComponents(ctx, factory)
	if err != nil {
		return err
	}

	err = app.builtExtensions.NotifyPipelineReady()
	if err != nil {
		return err
	}

	// Everything is ready, now run until an event requiring shutdown happens.
	app.runAndWaitForShutdownEvent()

	// Accumulate errors and proceed with shutting down remaining components.
	var errs []error

	// Begin shutdown sequence.
	runtime.KeepAlive(ballast)
	app.logger.Info("Starting shutdown...")

	err = app.builtExtensions.NotifyPipelineNotReady()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to notify that pipeline is not ready: %w", err))
	}

	err = app.shutdownPipelines(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown pipelines: %w", err))
	}

	err = app.shutdownExtensions(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown extensions: %w", err))
	}

	err = applicationTelemetry.shutdown()
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to shutdown extensions: %w", err))
	}

	app.logger.Info("Shutdown complete.")
	app.stateChannel <- Closed
	close(app.stateChannel)

	return componenterror.CombineErrors(errs)
}

// Run starts the collector according to the command and configuration
// given by the user, and waits for it to complete.
func (app *Application) Run() error {
	// From this point on do not show usage in case of error.
	app.rootCmd.SilenceUsage = true

	return app.rootCmd.Execute()
}

const (
	zPipelineName  = "zpipelinename"
	zComponentName = "zcomponentname"
	zComponentKind = "zcomponentkind"
	zExtensionName = "zextensionname"
)

func (app *Application) handleServicezRequest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	internal.WriteHTMLHeader(w, internal.HeaderData{Title: "Service"})
	internal.WriteHTMLComponentHeader(w, internal.ComponentHeaderData{
		Name:              "Pipelines",
		ComponentEndpoint: pipelinezPath,
		Link:              true,
	})
	internal.WriteHTMLComponentHeader(w, internal.ComponentHeaderData{
		Name:              "Extensions",
		ComponentEndpoint: extensionzPath,
		Link:              true,
	})
	internal.WriteHTMLPropertiesTable(w, internal.PropertiesTableData{Name: "Build And Runtime", Properties: version.InfoVar})
	internal.WriteHTMLFooter(w)
}

func (app *Application) handlePipelinezRequest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	pipelineName := r.Form.Get(zPipelineName)
	componentName := r.Form.Get(zComponentName)
	componentKind := r.Form.Get(zComponentKind)
	internal.WriteHTMLHeader(w, internal.HeaderData{Title: "Pipelines"})
	internal.WriteHTMLPipelinesSummaryTable(w, app.getPipelinesSummaryTableData())
	if pipelineName != "" && componentName != "" && componentKind != "" {
		fullName := componentName
		if componentKind == "processor" {
			fullName = pipelineName + "/" + componentName
		}
		internal.WriteHTMLComponentHeader(w, internal.ComponentHeaderData{
			Name: componentKind + ": " + fullName,
		})
		// TODO: Add config + status info.
	}
	internal.WriteHTMLFooter(w)
}

func (app *Application) handleExtensionzRequest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	extensionName := r.Form.Get(zExtensionName)
	internal.WriteHTMLHeader(w, internal.HeaderData{Title: "Extensions"})
	internal.WriteHTMLExtensionsSummaryTable(w, app.getExtensionsSummaryTableData())
	if extensionName != "" {
		internal.WriteHTMLComponentHeader(w, internal.ComponentHeaderData{
			Name: extensionName,
		})
		// TODO: Add config + status info.
	}
	internal.WriteHTMLFooter(w)
}

func (app *Application) getPipelinesSummaryTableData() internal.SummaryPipelinesTableData {
	data := internal.SummaryPipelinesTableData{
		ComponentEndpoint: pipelinezPath,
	}

	data.Rows = make([]internal.SummaryPipelinesTableRowData, 0, len(app.builtExtensions))
	for c, p := range app.builtPipelines {
		row := internal.SummaryPipelinesTableRowData{
			FullName:            c.Name,
			InputType:           string(c.InputType),
			MutatesConsumedData: p.MutatesConsumedData,
			Receivers:           c.Receivers,
			Processors:          c.Processors,
			Exporters:           c.Exporters,
		}
		data.Rows = append(data.Rows, row)
	}

	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].FullName < data.Rows[j].FullName
	})
	return data
}

func (app *Application) getExtensionsSummaryTableData() internal.SummaryExtensionsTableData {
	data := internal.SummaryExtensionsTableData{
		ComponentEndpoint: extensionzPath,
	}

	data.Rows = make([]internal.SummaryExtensionsTableRowData, 0, len(app.builtExtensions))
	for c := range app.builtExtensions {
		row := internal.SummaryExtensionsTableRowData{FullName: c.Name()}
		data.Rows = append(data.Rows, row)
	}

	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].FullName < data.Rows[j].FullName
	})
	return data
}

func (app *Application) createMemoryBallast() ([]byte, uint64) {
	ballastSizeMiB := builder.MemBallastSize()
	if ballastSizeMiB > 0 {
		ballastSizeBytes := uint64(ballastSizeMiB) * 1024 * 1024
		ballast := make([]byte, ballastSizeBytes)
		app.logger.Info("Using memory ballast", zap.Int("MiBs", ballastSizeMiB))
		return ballast, ballastSizeBytes
	}
	return nil, 0
}
