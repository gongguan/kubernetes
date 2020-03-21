/*
Copyright 2016 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubelet

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	cadvisorapiv1 "github.com/google/cadvisor/info/v1"
	"k8s.io/klog"
	"k8s.io/utils/mount"
	utilpath "k8s.io/utils/path"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/kubelet/cm"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	kubelettypes "k8s.io/kubernetes/pkg/kubelet/types"
)

// GetPods returns all pods bound to the kubelet and their spec, and the mirror
// pods.
func (kl *Kubelet) GetPods() []*v1.Pod {
	pods := kl.podManager.GetPods()
	// a kubelet running without apiserver requires an additional
	// update of the static pod status. See #57106
	for _, p := range pods {
		if kubelettypes.IsStaticPod(p) {
			if status, ok := kl.statusManager.GetPodStatus(p.UID); ok {
				klog.V(2).Infof("status for pod %v updated to %v", p.Name, status)
				p.Status = status
			}
		}
	}
	return pods
}

// GetRunningPods returns all pods running on kubelet from looking at the
// container runtime cache. This function converts kubecontainer.Pod to
// v1.Pod, so only the fields that exist in both kubecontainer.Pod and
// v1.Pod are considered meaningful.
func (kl *Kubelet) GetRunningPods() ([]*v1.Pod, error) {
	pods, err := kl.runtimeCache.GetPods()
	if err != nil {
		return nil, err
	}

	apiPods := make([]*v1.Pod, 0, len(pods))
	for _, pod := range pods {
		apiPods = append(apiPods, pod.ToAPIPod())
	}
	return apiPods, nil
}

// GetPodByFullName gets the pod with the given 'full' name, which
// incorporates the namespace as well as whether the pod was found.
func (kl *Kubelet) GetPodByFullName(podFullName string) (*v1.Pod, bool) {
	return kl.podManager.GetPodByFullName(podFullName)
}

// GetPodByName provides the first pod that matches namespace and name, as well
// as whether the pod was found.
func (kl *Kubelet) GetPodByName(namespace, name string) (*v1.Pod, bool) {
	return kl.podManager.GetPodByName(namespace, name)
}

// GetPodByCgroupfs provides the pod that maps to the specified cgroup, as well
// as whether the pod was found.
func (kl *Kubelet) GetPodByCgroupfs(cgroupfs string) (*v1.Pod, bool) {
	pcm := kl.containerManager.NewPodContainerManager()
	if result, podUID := pcm.IsPodCgroup(cgroupfs); result {
		return kl.podManager.GetPodByUID(podUID)
	}
	return nil, false
}

// TODO remove it when kubelet no more injected to ResourceAnalyzer.
// GetNode returns the node info for the configured node name of this Kubelet.
func (kl *Kubelet) GetNode() (*v1.Node, error) {
	return kl.basicInfo.GetNode()
}

// getRuntime returns the current Runtime implementation in use by the kubelet.
func (kl *Kubelet) getRuntime() kubecontainer.Runtime {
	return kl.containerRuntime
}

// GetNodeConfig returns the container manager node config.
func (kl *Kubelet) GetNodeConfig() cm.NodeConfig {
	return kl.containerManager.GetNodeConfig()
}

// GetPodCgroupRoot returns the listeral cgroupfs value for the cgroup containing all pods
func (kl *Kubelet) GetPodCgroupRoot() string {
	return kl.containerManager.GetPodCgroupRoot()
}

// getPodVolumePathListFromDisk returns a list of the volume paths by reading the
// volume directories for the given pod from the disk.
func (kl *Kubelet) getPodVolumePathListFromDisk(podUID types.UID) ([]string, error) {
	volumes := []string{}
	podVolDir := kl.basicInfo.GetPodVolumesDir(podUID)

	if pathExists, pathErr := mount.PathExists(podVolDir); pathErr != nil {
		return volumes, fmt.Errorf("error checking if path %q exists: %v", podVolDir, pathErr)
	} else if !pathExists {
		klog.Warningf("Path %q does not exist", podVolDir)
		return volumes, nil
	}

	volumePluginDirs, err := ioutil.ReadDir(podVolDir)
	if err != nil {
		klog.Errorf("Could not read directory %s: %v", podVolDir, err)
		return volumes, err
	}
	for _, volumePluginDir := range volumePluginDirs {
		volumePluginName := volumePluginDir.Name()
		volumePluginPath := filepath.Join(podVolDir, volumePluginName)
		volumeDirs, err := utilpath.ReadDirNoStat(volumePluginPath)
		if err != nil {
			return volumes, fmt.Errorf("could not read directory %s: %v", volumePluginPath, err)
		}
		for _, volumeDir := range volumeDirs {
			volumes = append(volumes, filepath.Join(volumePluginPath, volumeDir))
		}
	}
	return volumes, nil
}

func (kl *Kubelet) getMountedVolumePathListFromDisk(podUID types.UID) ([]string, error) {
	mountedVolumes := []string{}
	volumePaths, err := kl.getPodVolumePathListFromDisk(podUID)
	if err != nil {
		return mountedVolumes, err
	}
	for _, volumePath := range volumePaths {
		isNotMount, err := kl.mounter.IsLikelyNotMountPoint(volumePath)
		if err != nil {
			return mountedVolumes, err
		}
		if !isNotMount {
			mountedVolumes = append(mountedVolumes, volumePath)
		}
	}
	return mountedVolumes, nil
}

// podVolumesSubpathsDirExists returns true if the pod volume-subpaths directory for
// a given pod exists
func (kl *Kubelet) podVolumeSubpathsDirExists(podUID types.UID) (bool, error) {
	podVolDir := kl.basicInfo.GetPodVolumeSubpathsDir(podUID)

	if pathExists, pathErr := mount.PathExists(podVolDir); pathErr != nil {
		return true, fmt.Errorf("error checking if path %q exists: %v", podVolDir, pathErr)
	} else if !pathExists {
		return false, nil
	}
	return true, nil
}

// GetVersionInfo returns information about the version of cAdvisor in use.
func (kl *Kubelet) GetVersionInfo() (*cadvisorapiv1.VersionInfo, error) {
	return kl.cadvisor.VersionInfo()
}

// GetCachedMachineInfo assumes that the machine info can't change without a reboot
func (kl *Kubelet) GetCachedMachineInfo() (*cadvisorapiv1.MachineInfo, error) {
	return kl.machineInfo, nil
}
