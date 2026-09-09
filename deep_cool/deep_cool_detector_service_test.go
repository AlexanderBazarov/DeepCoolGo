package deepcool

import (
	"testing"

	"github.com/google/gousb"
)

func TestIsCompatibleDeepCoolDevice(t *testing.T) {
	tests := []struct {
		name   string
		change func(*gousb.DeviceDesc)
		want   bool
	}{
		{name: "bulk display", want: true},
		{
			name: "interrupt display",
			change: func(desc *gousb.DeviceDesc) {
				endpoints := desc.Configs[1].Interfaces[0].AltSettings[0].Endpoints
				ep := endpoints[1]
				ep.TransferType = gousb.TransferTypeInterrupt
				endpoints[1] = ep
			},
			want: true,
		},
		{
			name:   "different vendor",
			change: func(desc *gousb.DeviceDesc) { desc.Vendor = 0x1234 },
		},
		{
			name:   "unsupported DeepCool product",
			change: func(desc *gousb.DeviceDesc) { desc.Product = 0xffff },
		},
		{
			name:   "missing configuration",
			change: func(desc *gousb.DeviceDesc) { delete(desc.Configs, 1) },
		},
		{
			name: "different interface",
			change: func(desc *gousb.DeviceDesc) {
				desc.Configs[1].Interfaces[0].Number = 1
			},
		},
		{
			name: "different alternate setting",
			change: func(desc *gousb.DeviceDesc) {
				desc.Configs[1].Interfaces[0].AltSettings[0].Alternate = 1
			},
		},
		{
			name: "missing output endpoint",
			change: func(desc *gousb.DeviceDesc) {
				desc.Configs[1].Interfaces[0].AltSettings[0].Endpoints = nil
			},
		},
		{
			name: "input endpoint only",
			change: func(desc *gousb.DeviceDesc) {
				endpoints := desc.Configs[1].Interfaces[0].AltSettings[0].Endpoints
				ep := endpoints[1]
				ep.Address = 0x81
				ep.Direction = gousb.EndpointDirectionIn
				delete(endpoints, 1)
				endpoints[ep.Address] = ep
			},
		},
		{
			name: "unsupported transfer type",
			change: func(desc *gousb.DeviceDesc) {
				endpoints := desc.Configs[1].Interfaces[0].AltSettings[0].Endpoints
				ep := endpoints[1]
				ep.TransferType = gousb.TransferTypeIsochronous
				endpoints[1] = ep
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			desc := &gousb.DeviceDesc{
				Vendor:  gousb.ID(VendorID),
				Product: 0x0026,
				Configs: map[int]gousb.ConfigDesc{
					1: {
						Number: 1,
						Interfaces: []gousb.InterfaceDesc{
							{
								Number: 0,
								AltSettings: []gousb.InterfaceSetting{
									{
										Number:    0,
										Alternate: 0,
										Endpoints: map[gousb.EndpointAddress]gousb.EndpointDesc{
											1: {
												Address:      1,
												Number:       1,
												Direction:    gousb.EndpointDirectionOut,
												TransferType: gousb.TransferTypeBulk,
											},
										},
									},
								},
							},
						},
					},
				},
			}
			if test.change != nil {
				test.change(desc)
			}
			if got := isCompatibleDeepCoolDevice(desc); got != test.want {
				t.Fatalf("isCompatibleDeepCoolDevice() = %v, want %v", got, test.want)
			}
		})
	}

	if isCompatibleDeepCoolDevice(nil) {
		t.Fatal("nil descriptor is compatible")
	}
}
