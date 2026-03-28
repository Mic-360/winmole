package services

type Container struct {
	Config      *ConfigService
	Logger      *Logger
	Runtime     *RuntimeService
	Cleaner     *CleanerService
	Uninstaller *UninstallService
	Optimizer   *OptimizerService
	Purger      *PurgeService
}

func NewContainer() (*Container, error) {
	config, err := NewConfigService()
	if err != nil {
		return nil, err
	}
	logger, err := NewLogger(config.LogsDir())
	if err != nil {
		return nil, err
	}
	container := &Container{Config: config, Logger: logger}
	container.Runtime = NewRuntimeService(logger)
	container.Cleaner = NewCleanerService(logger)
	container.Uninstaller = NewUninstallService(logger)
	container.Optimizer = NewOptimizerService(logger)
	container.Purger = NewPurgeService(logger)
	logger.Info("app", "service container initialized")
	return container, nil
}

func (c *Container) Close() error {
	if c == nil || c.Logger == nil {
		return nil
	}
	return c.Logger.Close()
}
