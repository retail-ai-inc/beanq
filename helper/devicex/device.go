package devicex

import (
	"errors"
	lnet "net"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"golang.org/x/sync/errgroup"
)

// DeviceI
// @Description:
type DeviceI interface {
	Info() error
}

var _ DeviceI = new(device)

type device struct {
	Memory devMemory `json:"memory"`
	Cpu    devCpu    `json:"cpu"`
	Disk   devDisk   `json:"disk"`
	Net    devNet    `json:"net"`
}

var Device = new(device)

func (t *device) Info() error {

	eg := new(errgroup.Group)
	//memory information
	eg.Go(func() error {
		return t.memory()
	})
	//disk information
	eg.Go(func() error {
		return t.disk()
	})
	//cpu information
	eg.Go(func() error {
		return t.cpu()
	})
	eg.Go(func() error {
		return t.net()
	})
	return eg.Wait()
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
func (t *device) disk() error {
	dk, err := disk.Partitions(false)
	if err != nil {
		return err
	}
	if len(dk) <= 0 {
		return errors.New("1111")
	}

	for _, stat := range dk {
		if stat.Mountpoint != "/" {
			continue
		}
		s, err := disk.Usage(stat.Mountpoint)
		if err != nil {
			return err
		}
		if s != nil {
			t.Disk = devDisk{
				Name:           stat.Device,
				AvailableBytes: s.Free,
				UsageBytes:     s.Used,
				UsageRatio:     s.UsedPercent,
			}
		}
		break
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
