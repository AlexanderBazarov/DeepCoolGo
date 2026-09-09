package monitor

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type PowerMonitor struct {
	prevEnergy uint64
	prevTime   time.Time
}

func NewPowerMonitor() *PowerMonitor {
	return &PowerMonitor{
		prevEnergy: 0,
		prevTime:   time.Now(),
	}
}

func readPower() (uint64, error) {
	powerRow, err := os.ReadFile("/sys/class/powercap/intel-rapl:0/energy_uj")
	if err != nil {
		return 0, err
	}

	power, err := strconv.ParseUint(
		strings.TrimSpace(string(powerRow)),
		10,
		64,
	)
	if err != nil {
		return 0, err
	}

	return power, nil
}

func (p *PowerMonitor) Power() (float64, error) {
	currentEnergy, err := readPower()
	if err != nil {
		return 0, err
	}

	now := time.Now()

	if p.prevEnergy == 0 {
		p.prevEnergy = currentEnergy
		return 0, err
	}

	deltaEnergy := currentEnergy - p.prevEnergy
	deltaTime := now.Sub(p.prevTime).Seconds()

	p.prevEnergy = currentEnergy
	p.prevTime = now

	if deltaTime <= 0 {
		return 0, nil
	}

	return (float64(deltaEnergy) / 1_000_000) / deltaTime, nil
}
