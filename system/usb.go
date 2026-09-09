package system

import (
	"fmt"

	"github.com/google/gousb"
)

type USBService struct {
	ctx  *gousb.Context
	dev  *gousb.Device
	cfg  *gousb.Config
	intf *gousb.Interface
	ep   *gousb.OutEndpoint
}

func NewUSBService(device USBDevice) (*USBService, error) {
	service := &USBService{}

	if err := service.InitNewUSBSession(device); err != nil {
		return nil, err
	}

	return service, nil
}

func (d *USBService) InitNewUSBSession(device USBDevice) error {
	d.Close()
	d.ctx = gousb.NewContext()

	devices, err := d.ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
		return desc.Vendor == gousb.ID(device.VendorID) &&
			desc.Product == gousb.ID(device.ProductID) &&
			desc.Bus == device.Bus && desc.Address == device.Address
	})
	if len(devices) == 0 {
		d.Close()
		if err != nil {
			return fmt.Errorf("open USB device: %w", err)
		}
		return fmt.Errorf(
			"USB device %04x:%04x bus=%d addr=%d not found",
			device.VendorID,
			device.ProductID,
			device.Bus,
			device.Address,
		)
	}

	dev := devices[0]
	for _, other := range devices[1:] {
		other.Close()
	}
	d.dev = dev

	fmt.Printf(
		"Found vid=%04x pid=%04x bus=%d addr=%d\n",
		dev.Desc.Vendor,
		dev.Desc.Product,
		dev.Desc.Bus,
		dev.Desc.Address,
	)

	if err := d.dev.SetAutoDetach(true); err != nil {
		d.Close()
		return fmt.Errorf("set auto detach: %w", err)
	}

	cfg, err := d.dev.Config(1)
	if err != nil {
		d.Close()
		return fmt.Errorf("open config: %w", err)
	}

	d.cfg = cfg

	intf, err := d.cfg.Interface(0, 0)
	if err != nil {
		d.Close()
		return fmt.Errorf("claim interface: %w", err)
	}

	d.intf = intf

	ep, err := d.intf.OutEndpoint(1)
	if err != nil {
		d.Close()
		return fmt.Errorf("open OUT endpoint: %w", err)
	}

	d.ep = ep

	return nil
}

func (d *USBService) WriteBytes(data []byte) (int, error) {
	if d.ep == nil {
		return 0, fmt.Errorf("USB connection is not initialized")
	}

	n, err := d.ep.Write(data)
	if err != nil {
		return n, fmt.Errorf("USB write failed: %w", err)
	}

	fmt.Printf("USB: wrote %d bytes\n", n)

	return n, nil
}

func (d *USBService) Close() {
	d.ep = nil

	if d.intf != nil {
		d.intf.Close()
		d.intf = nil
	}

	if d.cfg != nil {
		d.cfg.Close()
		d.cfg = nil
	}

	if d.dev != nil {
		d.dev.Close()
		d.dev = nil
	}

	if d.ctx != nil {
		d.ctx.Close()
		d.ctx = nil
	}
}
