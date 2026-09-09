package deepcool

import (
	"errors"
	"fmt"

	"Lm360Go/system"
)

func NewDeepCoolService() (*DeepCoolService, error) {
	detector := NewDeepCoolDetectorService()
	devices, err := detector.Detect()
	if len(devices) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("compatible DeepCool displays not found")
	}

	for _, device := range devices {
		usb, openErr := system.NewUSBService(device)
		if openErr == nil {
			return &DeepCoolService{usb: usb}, nil
		}
		err = errors.Join(err, fmt.Errorf("DeepCool display bus=%d addr=%d: %w", device.Bus, device.Address, openErr))
	}
	return nil, fmt.Errorf("open compatible DeepCool display: %w", err)
}

type DeepCoolService struct {
	usb *system.USBService
}

func (d *DeepCoolService) PrintFrame(frame Frame) error {
	_, err := d.usb.WriteBytes(frameHeader)
	if err != nil {
		return err
	}

	_, err = d.usb.WriteBytes(frame.render())
	if err != nil {
		return err
	}
	fmt.Printf("frame: wroted \n")
	return nil
}

func (d *DeepCoolService) Close() {
	d.usb.Close()
}
