package deepcool

import (
	"fmt"
	"slices"
	"sort"

	"github.com/google/gousb"

	"Lm360Go/system"
)

const VendorID int = 0x3633

var compatibleProductIDs = []gousb.ID{
	0x0026,
}

type DeepCoolDetectorService struct{}

func NewDeepCoolDetectorService() *DeepCoolDetectorService {
	return &DeepCoolDetectorService{}
}

func (d *DeepCoolDetectorService) Detect() ([]system.USBDevice, error) {
	ctx := gousb.NewContext()
	defer ctx.Close()

	var devices []system.USBDevice
	_, err := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		if isCompatibleDeepCoolDevice(desc) {
			devices = append(devices, system.USBDevice{
				VendorID:  int(desc.Vendor),
				ProductID: int(desc.Product),
				Bus:       desc.Bus,
				Address:   desc.Address,
			})
		}
		return false
	})

	sort.Slice(devices, func(i, j int) bool {
		if devices[i].Bus == devices[j].Bus {
			return devices[i].Address < devices[j].Address
		}
		return devices[i].Bus < devices[j].Bus
	})

	if err != nil {
		return devices, fmt.Errorf("detect DeepCool displays: %w", err)
	}
	return devices, nil
}

func isCompatibleDeepCoolDevice(desc *gousb.DeviceDesc) bool {
	if desc == nil || desc.Vendor != gousb.ID(VendorID) || !slices.Contains(compatibleProductIDs, desc.Product) {
		return false
	}

	cfg, ok := desc.Configs[1]
	if !ok {
		return false
	}
	for _, intf := range cfg.Interfaces {
		if intf.Number != 0 {
			continue
		}
		for _, setting := range intf.AltSettings {
			if setting.Alternate != 0 {
				continue
			}
			ep, ok := setting.Endpoints[1]
			return ok && ep.Direction == gousb.EndpointDirectionOut &&
				(ep.TransferType == gousb.TransferTypeBulk || ep.TransferType == gousb.TransferTypeInterrupt)
		}
	}
	return false
}
