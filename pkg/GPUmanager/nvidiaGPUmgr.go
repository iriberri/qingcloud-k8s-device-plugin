package GPUmanager

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io/ioutil"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1alpha"
	"net"
	"os"
	"path"
	"regexp"
	"sync"
	"time"
)

const (
	// All NVIDIA GPUs cards should be mounted with nvidiactl and nvidia-uvm
	// If the driver installed correctly, these two devices will be there.
	nvidiaCtlDevice = "/dev/nvidiactl"
	nvidiaUVMDevice = "/dev/nvidia-uvm"
	// Optional device.
	nvidiaUVMToolsDevice = "/dev/nvidia-uvm-tools"
	devDirectory         = "/dev"
	nvidiaDeviceRE       = `^nvidia[0-9]*$`

	resourceName = "nvidia.com/gpu"
)

// nvidiaGPUManager manages nvidia gpu devices.
type nvidiaGPUManager struct {
	defaultDevices      []string
	devices             map[string]pluginapi.Device
	grpcServer          *grpc.Server
	hostPathPrefix      string
	containerPathPrefix string
}

func NewNvidiaGPUManager(hostPathPrefix, containerPathPrefix string) *nvidiaGPUManager {
	return &nvidiaGPUManager{
		devices:             make(map[string]pluginapi.Device),
		hostPathPrefix:      hostPathPrefix,
		containerPathPrefix: containerPathPrefix,
	}
}

// Discovers all NVIDIA GPU devices available on the local node by walking `/dev` directory.
func (ngm *nvidiaGPUManager) discoverGPUs() error {
	reg := regexp.MustCompile(nvidiaDeviceRE)
	files, err := ioutil.ReadDir(devDirectory)
	if err != nil {
		return err
	}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if reg.MatchString(f.Name()) {
			glog.Infof("Found Nvidia GPU %q\n", f.Name())
			ngm.devices[f.Name()] = pluginapi.Device{ID:f.Name(), Health:pluginapi.Healthy}
		}
	}
	return nil
}

func (ngm *nvidiaGPUManager) GetDeviceState(DeviceName string) string {
	return pluginapi.Healthy
}

// Discovers Nvidia GPU devices and sets up device access environment.
func (ngm *nvidiaGPUManager) Start() error {
	if _, err := os.Stat(nvidiaCtlDevice); err != nil {
		return err
	}

	if _, err := os.Stat(nvidiaUVMDevice); err != nil {
		return err
	}

	ngm.defaultDevices = []string{nvidiaCtlDevice, nvidiaUVMDevice}

	if _, err := os.Stat(nvidiaUVMToolsDevice); err == nil {
		ngm.defaultDevices = append(ngm.defaultDevices, nvidiaUVMToolsDevice)
	}

	if err := ngm.discoverGPUs(); err != nil {
		return err
	}

	return nil
}

// Implements DevicePlugin service functions
func (ngm *nvidiaGPUManager) ListAndWatch(emtpy *pluginapi.Empty, stream pluginapi.DevicePlugin_ListAndWatchServer) error {
	glog.Infoln("device-plugin: ListAndWatch start")
	changed := true
	for {
		for id, dev := range ngm.devices {
			state := ngm.GetDeviceState(id)
			if dev.Health != state {
				changed = true
				dev.Health = state
				ngm.devices[id] = dev
			}
		}
		if changed {
			resp := new(pluginapi.ListAndWatchResponse)
			for _, dev := range ngm.devices {
				resp.Devices = append(resp.Devices, &pluginapi.Device{ID:dev.ID, Health:dev.Health})
			}
			glog.Infof("ListAndWatch: send devices %v\n", resp)
			if err := stream.Send(resp); err != nil {
				glog.Errorf("device-plugin: cannot update device states: %v\n", err)
				ngm.grpcServer.Stop()
				return err
			}
		}
		changed = false
		time.Sleep(5 * time.Second)
	}
}

func (ngm *nvidiaGPUManager) Allocate(ctx context.Context, rqt *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	resp := new(pluginapi.AllocateResponse)
	// Add all requested devices to Allocate Response
	for _, id := range rqt.DevicesIDs {
		dev, ok := ngm.devices[id]
		if !ok {
			return nil, fmt.Errorf("invalid allocation request with non-existing device %s", id)
		}
		if dev.Health != pluginapi.Healthy {
			return nil, fmt.Errorf("invalid allocation request with unhealthy device %s", id)
		}
		resp.Devices = append(resp.Devices, &pluginapi.DeviceSpec{
			HostPath:      "/dev/" + id,
			ContainerPath: "/dev/" + id,
			Permissions:   "mrw",
		})
	}
	// Add all default devices to Allocate Response
	for _, d := range ngm.defaultDevices {
		resp.Devices = append(resp.Devices, &pluginapi.DeviceSpec{
			HostPath:      d,
			ContainerPath: d,
			Permissions:   "mrw",
		})
	}

	resp.Mounts = append(resp.Mounts, &pluginapi.Mount{
		ContainerPath: ngm.containerPathPrefix,
		HostPath:      ngm.hostPathPrefix,
		ReadOnly:      true,
	})
	return resp, nil
}

func (ngm *nvidiaGPUManager) Serve(pMountPath, kEndpoint, pEndpointPrefix string) {
	for {
		pluginEndpoint := fmt.Sprintf("%s-%d.sock", pEndpointPrefix, time.Now().Unix())
		pluginEndpointPath := path.Join(pMountPath, pluginEndpoint)
		glog.Infof("starting device-plugin server at: %s\n", pluginEndpointPath)
		lis, err := net.Listen("unix", pluginEndpointPath)
		if err != nil {
			glog.Fatalf("starting device-plugin server failed: %v", err)
		}
		ngm.grpcServer = grpc.NewServer()
		pluginapi.RegisterDevicePluginServer(ngm.grpcServer, ngm)

		var wg sync.WaitGroup
		wg.Add(1)
		// Starts device plugin service.
		go func() {
			defer wg.Done()
			// Blocking call to accept incoming connections.
			err := ngm.grpcServer.Serve(lis)
			glog.Errorf("device-plugin server stopped serving: %v", err)
		}()

		// Wait till the grpcServer is ready to serve services.
		for len(ngm.grpcServer.GetServiceInfo()) <= 0 {
			time.Sleep(1 * time.Second)
		}
		glog.Infoln("device-plugin server started serving")

		// Registers with Kubelet.
		err = Register(path.Join(pMountPath, kEndpoint), pluginEndpoint, resourceName)
		if err != nil {
			ngm.grpcServer.Stop()
			wg.Wait()
			glog.Fatal(err)
		}
		glog.Infoln("device-plugin registered with the kubelet")

		// This is checking if the plugin socket was deleted. If so,
		// stop the grpc server and start the whole thing again.
		for {
			if _, err := os.Lstat(pluginEndpointPath); err != nil {
				glog.Errorln(err)
				ngm.grpcServer.Stop()
				break
			}
			time.Sleep(1 * time.Second)
		}
		wg.Wait()
	}
}

// Act as a grpc client and register with the kubelet.
func Register(kubeletEndpoint, pluginEndpoint, resourceName string) error {
	conn, err := grpc.Dial(kubeletEndpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		return fmt.Errorf("device-plugin: cannot connect to kubelet service: %v", err)
	}
	defer conn.Close()
	client := pluginapi.NewRegistrationClient(conn)

	request := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     pluginEndpoint,
		ResourceName: resourceName,
	}

	if _, err = client.Register(context.Background(), request); err != nil {
		return fmt.Errorf("device-plugin: cannot register to kubelet service: %v", err)
	}
	return nil
}
