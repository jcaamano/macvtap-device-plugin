package macvtap

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/vishvananda/netlink"
	"golang.org/x/net/context"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	tapPath         = "/dev/tap"
	suffix          = "Mvp"
	defaultCapacity = 100
)

type MacvtapDevicePlugin struct {
	Name     string
	Master   string
	Mode     string
	Capacity int
}

func (mdp *MacvtapDevicePlugin) generateMacvtapDevices() []*pluginapi.Device {
	var macvtapDevs []*pluginapi.Device

	var capacity = mdp.Capacity
	if capacity <= 0 {
		capacity = defaultCapacity
	}

	for i := 0; i < capacity; i++ {
		name := fmt.Sprint(mdp.Name, suffix, i)
		macvtapDevs = append(macvtapDevs, &pluginapi.Device{
			ID:     name,
			Health: pluginapi.Healthy,
		})
	}
	return macvtapDevs
}

func masterExists(master string) bool {
	_, err := netlink.LinkByName(master)
	if err != nil {
		glog.V(3).Infof("Master %s not found: %v", master, err)
		return false
	}
	// TODO check more details about master ?
	return true
}

func modeFromString(s string) (netlink.MacvlanMode, error) {
	switch s {
	case "", "bridge":
		return netlink.MACVLAN_MODE_BRIDGE, nil
	case "private":
		return netlink.MACVLAN_MODE_PRIVATE, nil
	case "vepa":
		return netlink.MACVLAN_MODE_VEPA, nil
	default:
		return 0, fmt.Errorf("unknown macvtap mode: %q", s)
	}
}

func (mdp *MacvtapDevicePlugin) createMacvtap(name string) (int, error) {
	// attempt to delete any previously existing link
	if l, _ := netlink.LinkByName(name); l != nil {
		_ = netlink.LinkDel(l)
	}

	m, err := netlink.LinkByName(mdp.Master)
	if err != nil {
		return 0, fmt.Errorf("failed to lookup master %q: %v", mdp.Master, err)
	}

	mode, err := modeFromString(mdp.Mode)
	if err != nil {
		return 0, err
	}

	mv := &netlink.Macvtap{
		Macvlan: netlink.Macvlan{
			LinkAttrs: netlink.LinkAttrs{
				Name:        name,
				ParentIndex: m.Attrs().Index,
				// we had crashes if we did not set txqlen to some value
				TxQLen: m.Attrs().TxQLen,
			},
			Mode: mode,
		},
	}

	if err := netlink.LinkAdd(mv); err != nil {
		return 0, fmt.Errorf("failed to create macvtap: %v", err)
	}

	if err := netlink.LinkSetUp(mv); err != nil {
		return 0, fmt.Errorf("failed to set %q UP: %v", name, err)
	}

	return mv.Attrs().Index, nil
}

func (mdp *MacvtapDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	masterDevs := mdp.generateMacvtapDevices()
	noMasterDevs := make([]*pluginapi.Device, 0)
	emitResponse := func(masterExists bool) {
		if masterExists {
			glog.V(3).Info("Master exists, sending ListAndWatch response with available devices")
			s.Send(&pluginapi.ListAndWatchResponse{Devices: masterDevs})
		} else {
			glog.V(3).Info("Master does not exist, sending ListAndWatch response with no devices")
			s.Send(&pluginapi.ListAndWatchResponse{Devices: noMasterDevs})
		}
	}

	didMasterExist := masterExists(mdp.Master)
	emitResponse(didMasterExist)

	for {
		doesMasterExist := masterExists(mdp.Master)
		if didMasterExist != doesMasterExist {
			emitResponse(doesMasterExist)
			didMasterExist = doesMasterExist
		}
		time.Sleep(10 * time.Second)
	}
}

func (mdp *MacvtapDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	var response pluginapi.AllocateResponse

	for _, req := range r.ContainerRequests {
		var devices []*pluginapi.DeviceSpec
		for _, vnic := range req.DevicesIDs {
			dev := new(pluginapi.DeviceSpec)
			index, err := mdp.createMacvtap(vnic)
			if err != nil {
				return nil, err
			}
			devPath := fmt.Sprint(tapPath, index)
			dev.HostPath = devPath
			dev.ContainerPath = devPath
			dev.Permissions = "rw"
			devices = append(devices, dev)
		}

		response.ContainerResponses = append(response.ContainerResponses, &pluginapi.ContainerAllocateResponse{
			Devices: devices,
		})
	}

	return &response, nil
}

func (mdp *MacvtapDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return nil, nil
}

func (mdp *MacvtapDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return nil, nil
}
