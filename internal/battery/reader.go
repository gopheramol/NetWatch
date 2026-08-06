// Package battery reads the host's local battery/UPS charge level from
// Linux sysfs. It is intentionally best-effort: a host with no battery
// (the common case for a home server) simply reports Present=false, never
// an error, so callers can treat "no battery" as a normal, silent no-op.
package battery

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const powerSupplyPath = "/sys/class/power_supply"

// Reading is a single battery/UPS charge snapshot.
type Reading struct {
	Present  bool
	Percent  float64
	Charging bool
}

// Reader reads the current battery state.
type Reader interface {
	Read(ctx context.Context) (Reading, error)
}

type linuxReader struct{}

// NewLinuxReader builds a Reader backed by /sys/class/power_supply, the
// standard Linux sysfs interface for batteries and UPS devices reporting
// capacity (used by upower, acpi, and most NUT/UPS drivers alike).
func NewLinuxReader() Reader {
	return &linuxReader{}
}

func (r *linuxReader) Read(_ context.Context) (Reading, error) {
	entries, err := os.ReadDir(powerSupplyPath)
	if err != nil {
		return Reading{Present: false}, nil
	}

	for _, entry := range entries {
		devicePath := filepath.Join(powerSupplyPath, entry.Name())

		typeBytes, err := os.ReadFile(filepath.Join(devicePath, "type"))
		if err != nil || strings.TrimSpace(string(typeBytes)) != "Battery" {
			continue
		}

		capacityBytes, err := os.ReadFile(filepath.Join(devicePath, "capacity"))
		if err != nil {
			continue
		}
		percent, err := strconv.ParseFloat(strings.TrimSpace(string(capacityBytes)), 64)
		if err != nil {
			continue
		}

		statusBytes, _ := os.ReadFile(filepath.Join(devicePath, "status"))
		status := strings.TrimSpace(string(statusBytes))
		charging := status == "Charging" || status == "Full"

		return Reading{Present: true, Percent: percent, Charging: charging}, nil
	}

	return Reading{Present: false}, nil
}
