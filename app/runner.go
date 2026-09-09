package app

import (
	"context"
	"log/slog"
	"time"

	deepcool "Lm360Go/deep_cool"
	"Lm360Go/monitor"
	"Lm360Go/themes"
)

type Runner struct {
	collector *monitor.Collector
	theme     themes.Theme
	display   *deepcool.DeepCoolService
	interval  time.Duration
	log       *slog.Logger
}

func NewRunner(
	collector *monitor.Collector,
	theme themes.Theme,
	display *deepcool.DeepCoolService,
	interval time.Duration,
	log *slog.Logger,
) *Runner {
	return &Runner{
		collector: collector,
		theme:     theme,
		display:   display,
		interval:  interval,
		log:       log,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	var prev *themes.CPUData

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			data, err := r.collector.Collect()
			if err != nil {
				r.log.Warn("collect metrics", "err", err)
				continue
			}

			if prev != nil && *prev == data {
				continue
			}

			r.log.Debug("metrics",
				"cpu", data.CPU,
				"power", data.Power,
				"temp", data.Temp,
				"ghz", data.GHz,
			)

			if err := r.display.PrintFrame(r.frame(data)); err != nil {
				r.log.Error("print frame", "err", err)
				continue
			}

			snapshot := data
			prev = &snapshot
		}
	}
}

func (r *Runner) frame(data themes.CPUData) deepcool.Frame {
	frame := deepcool.Frame{}
	frame.AddAll(r.theme.Screen(data))

	return frame
}
