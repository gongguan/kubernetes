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

	"k8s.io/klog"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
)

const (
	// KubeMarkMasqChain is the mark-for-masquerade chain
	// TODO: clean up this logic in kube-proxy
	KubeMarkMasqChain utiliptables.Chain = "KUBE-MARK-MASQ"

	// KubeMarkDropChain is the mark-for-drop chain
	KubeMarkDropChain utiliptables.Chain = "KUBE-MARK-DROP"

	// KubePostroutingChain is kubernetes postrouting rules
	KubePostroutingChain utiliptables.Chain = "KUBE-POSTROUTING"

	// KubeFirewallChain is kubernetes firewall rules
	KubeFirewallChain utiliptables.Chain = "KUBE-FIREWALL"
)

// updatePodCIDR updates the pod CIDR in the runtime state if it is different
// from the current CIDR. Return true if pod CIDR is actually changed.
func (kl *Kubelet) updatePodCIDR(cidr string) (bool, error) {
	kl.updatePodCIDRMux.Lock()
	defer kl.updatePodCIDRMux.Unlock()

	podCIDR := kl.runtimeState.podCIDR()

	if podCIDR == cidr {
		return false, nil
	}

	// kubelet -> generic runtime -> runtime shim -> network plugin
	// docker/non-cri implementations have a passthrough UpdatePodCIDR
	if err := kl.getRuntime().UpdatePodCIDR(cidr); err != nil {
		// If updatePodCIDR would fail, theoretically pod CIDR could not change.
		// But it is better to be on the safe side to still return true here.
		return true, fmt.Errorf("failed to update pod CIDR: %v", err)
	}

	klog.Infof("Setting Pod CIDR: %v -> %v", podCIDR, cidr)
	kl.runtimeState.setPodCIDR(cidr)
	return true, nil
}
