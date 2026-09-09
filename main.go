package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"Lm360Go/app"
	"Lm360Go/config"
	deepcool "Lm360Go/deep_cool"
	"Lm360Go/monitor"
	"Lm360Go/themes"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	if err := run(log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	configPath, err := config.DefaultPath()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	log.Info("config loaded",
		"path", configPath,
		"theme", cfg.Theme,
		"themes_path", cfg.ThemesPath,
		"refresh_rate", cfg.RefreshRate,
	)

	theme, err := themes.NewThemeLoaderService().Load(cfg.ThemePath())
	if err != nil {
		return fmt.Errorf("load theme %s: %w", cfg.ThemePath(), err)
	}

	display, err := deepcool.NewDeepCoolService()
	if err != nil {
		return fmt.Errorf("open display: %w", err)
	}
	defer display.Close()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	runner := app.NewRunner(
		monitor.NewCollector(),
		theme,
		display,
		cfg.Interval(),
		log,
	)

	log.Info("started", "interval", cfg.Interval())

	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	log.Info("shutting down")

	return nil
}
