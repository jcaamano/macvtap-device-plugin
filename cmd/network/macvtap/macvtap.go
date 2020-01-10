package main

import (
	"flag"
	"os"

	"github.com/golang/glog"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"github.com/kubevirt/kubernetes-device-plugins/pkg/network/macvtap"
)

func main() {
	flag.Parse()

	_, configDefined := os.LookupEnv(macvtap.ConfigEnvironmentVariable)
	if !configDefined {
		glog.Exitf("%s environment variable must be set", macvtap.ConfigEnvironmentVariable)
	}

	manager := dpm.NewManager(macvtap.MacvtapLister{})
	manager.Run()
}
