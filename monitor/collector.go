package monitor

import (
	"fmt"

	"Lm360Go/themes"
)

type Collector struct {
	power *PowerMonitor
	cpu   *CPUMonitor
}

func NewCollector() *Collector {
	return &Collector{
		power: NewPowerMonitor(),
		cpu:   NewCPUMonitor(),
	}
}

func (c *Collector) Collect() (themes.CPUData, error) {
	power, err := c.power.Power()
	if err != nil {
		return themes.CPUData{}, fmt.Errorf("read power: %w", err)
	}

	cpu, err := c.cpu.Usage()
	if err != nil {
		return themes.CPUData{}, fmt.Errorf("read cpu usage: %w", err)
	}

	temp, err := c.cpu.Temperature()
	if err != nil {
		return themes.CPUData{}, fmt.Errorf("read cpu temperature: %w", err)
	}

	ghz, err := c.cpu.FrequencyGHz()
	if err != nil {
		return themes.CPUData{}, fmt.Errorf("read cpu frequency: %w", err)
	}

	return themes.CPUData{
		CPU:   cpu,
		Power: power,
		GHz:   ghz,
		Temp:  temp,
	}, nil
}
