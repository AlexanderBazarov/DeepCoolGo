package monitor

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type CPUStat struct {
	Idle  uint64
	Total uint64
}

type CPUMonitor struct {
	prev CPUStat

	hwmonPath string
}

func NewCPUMonitor() *CPUMonitor {

	return &CPUMonitor{
		prev:      CPUStat{0, 0},
		hwmonPath: "",
	}
}

func (s *CPUMonitor) Usage() (float64, error) {
	current, err := readCPUUsage()
	if err != nil {
		return 0, err
	}

	totalDelta := current.Total - s.prev.Total
	idleDelta := current.Idle - s.prev.Idle

	s.prev = current

	if totalDelta == 0 {
		return 0, nil
	}

	usage := float64(totalDelta-idleDelta) /
		float64(totalDelta) * 100

	return math.Floor(usage), nil
}

func (s *CPUMonitor) Temperature() (float64, error) {
	if s.hwmonPath == "" {
		sensorPath, err := findCPUTempSensor()
		if err != nil {
			return 0, err
		}
		s.hwmonPath = sensorPath
	}
	data, err := os.ReadFile(s.hwmonPath)
	if err != nil {
		return 0, err
	}

	value, err := strconv.ParseFloat(
		strings.TrimSpace(string(data)),
		64,
	)
	if err != nil {
		return 0, err
	}

	return value / 1000.0, nil
}

func (s *CPUMonitor) FrequencyGHz() (float64, error) {
	paths, err := filepath.Glob(
		"/sys/devices/system/cpu/cpu[0-9]*/cpufreq/scaling_cur_freq",
	)
	if err != nil {
		return 0, err
	}

	var total float64
	var count int

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		khz, err := strconv.ParseFloat(
			strings.TrimSpace(string(data)),
			64,
		)
		if err != nil {
			continue
		}

		total += khz / 1_000_000.0
		count++
	}

	if count == 0 {
		return 0, os.ErrNotExist
	}

	return total / float64(count), nil
}

func readCPUUsage() (CPUStat, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return CPUStat{}, err
	}

	var (
		user    uint64
		nice    uint64
		system  uint64
		idle    uint64
		iowait  uint64
		irq     uint64
		softirq uint64
		steal   uint64
	)

	_, err = fmt.Sscanf(
		string(data),
		"cpu %d %d %d %d %d %d %d %d",
		&user,
		&nice,
		&system,
		&idle,
		&iowait,
		&irq,
		&softirq,
		&steal,
	)
	if err != nil {
		return CPUStat{}, err
	}

	idleAll := idle + iowait

	total := user +
		nice +
		system +
		idle +
		iowait +
		irq +
		softirq +
		steal

	return CPUStat{
		Idle:  idleAll,
		Total: total,
	}, nil
}

func findCPUTempSensor() (string, error) {
	hwmons, err := filepath.Glob("/sys/class/hwmon/hwmon*")
	if err != nil {
		return "", err
	}

	for _, hwmon := range hwmons {
		nameData, err := os.ReadFile(filepath.Join(hwmon, "name"))
		if err != nil {
			continue
		}

		name := strings.TrimSpace(string(nameData))

		var wantedLabel string

		switch name {
		case "k10temp":
			wantedLabel = "Tctl"

		case "coretemp":
			wantedLabel = "Package id 0"

		default:
			continue
		}

		labels, err := filepath.Glob(filepath.Join(hwmon, "temp*_label"))
		if err != nil {
			continue
		}

		for _, labelPath := range labels {
			data, err := os.ReadFile(labelPath)
			if err != nil {
				continue
			}

			if strings.TrimSpace(string(data)) != wantedLabel {
				continue
			}

			inputPath := strings.TrimSuffix(
				labelPath,
				"_label",
			) + "_input"

			return inputPath, nil
		}
	}

	return "", fmt.Errorf("CPU temperature sensor not found")
}
