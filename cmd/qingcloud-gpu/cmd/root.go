// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/gravitational/version"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/yunify/qingcloud-k8s-device-plugin/pkg/GPUmanager"

	"flag"
	"time"
)

const (
	// Device plugin settings.
	pluginMountPath      = "/var/lib/kubelet/device-plugins"
	kubeletEndpoint      = "kubelet.sock"
	pluginEndpointPrefix = "nvidiaGPU"
	paramHostPathName    = "host-path"
	paramContentPathName = "container-path"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "qingcloud-gpu",
	Short: "Kubernetes QingCloud GPU device plugin",
	Long:  `Device plugin to help kubernetes cluster utilize GPU resources on Qingcloud`,
	Run: func(cmd *cobra.Command, args []string) {
		pflag.Parse()
		versioninfo := version.Get()
		glog.Infof("gpumanager version: %s, commit id: %s, status: %s ", versioninfo.Version, versioninfo.GitCommit, versioninfo.GitTreeState)
		glog.Infoln("device-plugin started")
		ngm := GPUmanager.NewNvidiaGPUManager(viper.GetString(paramHostPathName), viper.GetString(paramContentPathName))
		// Keep on trying until success. This is required
		// because Nvidia drivers may not be installed initially.
		for {
			err := ngm.Start()
			if err == nil {
				break
			}
			// Use non-default level to avoid log spam.
			glog.V(3).Infof("nvidiaGPUManager.Start() failed: %v", err)
			time.Sleep(5 * time.Second)
		}
		ngm.Serve(pluginMountPath, kubeletEndpoint, pluginEndpointPrefix)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// when this action is called directly.
	rootCmd.Flags().AddGoFlagSet(flag.CommandLine)
	rootCmd.Flags().String(paramHostPathName, "/home/kubernetes/bin/nvidia", "Path on the host that contains nvidia libraries. This will be mounted inside the container as '-container-path'")
	rootCmd.Flags().String(paramContentPathName, "/usr/local/nvidia", "Path on the container that mounts '-host-path'")
}
