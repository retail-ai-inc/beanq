// MIT License

// Copyright The RAI Inc.
// The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package devicex

import (
	lnet "net"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type (
	DeviceI interface {
		Info() error
	}
	device struct {
		Memory devMemory `json:"memory"`
		Cpu    devCpu    `json:"cpu"`
		Net    devNet    `json:"net"`
	}
)

var (
	_      DeviceI = (*device)(nil)
	Device         = new(device)
)

func (t *device) Info() (err error) {

	if err = t.memory(); err != nil {
		return
	}
	if err = t.cpu(); err != nil {
		return
	}
	if err = t.net(); err != nil {
		return
	}
	return
}

func (t *device) cpu() error {
	coreCount, err := cpu.Counts(true)
	if err != nil {
		return err
	}
	t.Cpu.CoreCount = coreCount

	percents, err := cpu.Percent(200*time.Millisecond, false)
	if err != nil {
		return err
	}
	if len(percents) >= 1 {
		t.Cpu.Percent = percents[0]
	}
	return nil
}

func (t *device) memory() error {
	stat, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	t.Memory = devMemory{
		Total:       stat.Total,
		Available:   stat.Available,
		Used:        stat.Used,
		UsedPercent: stat.UsedPercent,
		Free:        stat.Free,
	}
	return nil
}

func (t *device) net() error {
	localIp, err := IpV4Addr()
	if err != nil {
		return err
	}
	ipnet := localIp
	var intf devNet
	stat, err := net.Interfaces()

	if err != nil {
		return err
	}
	for _, v := range stat {
		if len(v.Addrs) <= 0 {
			continue
		}
		for _, addr := range v.Addrs {
			if strings.Contains(addr.Addr, ipnet) {
				intf.Ip = ipnet
				intf.Mac = v.HardwareAddr
				intf.InterfacesName = v.Name
				break
			}
		}
	}
	t.Net = intf
	return nil
}

func IpV4Addr() (string, error) {
	addrs, err := lnet.InterfaceAddrs()

	if err != nil {
		return "", err
	}
	var ip string
	for _, address := range addrs {
		if ipnet, ok := address.(*lnet.IPNet); ok && !ipnet.IP.IsLoopback() {

			if ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
				break
			}
		}
	}
	return ip, nil
}
