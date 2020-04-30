/*
Copyright 2020 The Kubernetes Authors.

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

package nodeinfo

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/cloud-provider"
	cloudproviderapi "k8s.io/cloud-provider/api"
	"k8s.io/klog/v2"

	api "k8s.io/kubernetes/pkg/apis/core"
	k8s_api_v1 "k8s.io/kubernetes/pkg/apis/core/v1"
	utilnode "k8s.io/kubernetes/pkg/util/node"
	taintutil "k8s.io/kubernetes/pkg/util/taints"
	volutil "k8s.io/kubernetes/pkg/volume/util"
)

var _ Provider = &nodeInfo{}

// Provider defines the interface of kubelet node information.
type Provider interface {
	nodeInfoReader
	nodeInfoWritter
	GetHostIP() (net.IP, error)
	GetHostIPAnyWay() (net.IP, error)
	GetNodeAnyWay() (*v1.Node, error)
	GetNode() (*v1.Node, error)
	InitialNode(ctx context.Context) (*v1.Node, error)
	SetNodeStatus(node *v1.Node)
}

type nodeInfo struct {
	// hostname is the hostname the kubelet detected or was given via flag/config
	hostname string

	// hostnameOverridden indicates the hostname was overridden via flag/config
	hostnameOverridden bool

	nodeName types.NodeName

	// If non-nil, use this IP address for the node
	nodeIP net.IP

	clock clock.Clock

	// registerNode Set to true means node register itself with the apiserver.
	registerNode bool

	// for internal book keeping; access only from within registerWithApiserver
	registrationCompleted bool

	kubeClient clientset.Interface

	// a list of node labels to register
	nodeLabels map[string]string

	// Reference to this node.
	nodeRef *v1.ObjectReference

	nodeLister corelisters.NodeLister

	// If non-nil, this is a unique identifier for the node in an external database, eg. cloudprovider
	providerID string

	// Indicates that the node initialization happens in an external cloud controller
	externalCloudProvider bool

	cloud cloudprovider.Interface

	// Set to true to have the node register itself as schedulable.
	registerSchedulable bool
	// List of taints to add to a node object when the kubelet registers itself.
	registerWithTaints []api.Taint

	enableControllerAttachDetach bool

	// nodeStatusUpdateFrequency specifies how often kubelet computes node status. If node lease
	// feature is not enabled, it is also the frequency that kubelet posts node status to master.
	// In that case, be cautious when changing the constant, it must work with nodeMonitorGracePeriod
	// in nodecontroller. There are several constraints:
	// 1. nodeMonitorGracePeriod must be N times more than nodeStatusUpdateFrequency, where
	//    N means number of retries allowed for kubelet to post node status. It is pointless
	//    to make nodeMonitorGracePeriod be less than nodeStatusUpdateFrequency, since there
	//    will only be fresh values from Kubelet at an interval of nodeStatusUpdateFrequency.
	//    The constant must be less than podEvictionTimeout.
	// 2. nodeStatusUpdateFrequency needs to be large enough for kubelet to generate node
	//    status. Kubelet may fail to update node status reliably if the value is too small,
	//    as it takes time to gather all necessary node information.
	nodeStatusUpdateFrequency time.Duration

	// nodeStatusReportFrequency is the frequency that kubelet posts node
	// status to master. It is only used when node lease feature is enabled.
	nodeStatusReportFrequency time.Duration

	// lastStatusReportTime is the time when node status was last reported.
	lastStatusReportTime time.Time

	// handlers called during the tryUpdateNodeStatus cycle
	nodeStatusFuncs []func(*v1.Node) error

	// keepTerminatedPodVolumes is kubelet keepTerminatedPodVolumes.
	keepTerminatedPodVolumes bool
}

// NewNodeInfo creates nodeInfo Provider which provider node information.
func NewNodeInfo(
	hostname string,
	hostnameOverridden bool,
	nodeName types.NodeName,
	nodeIP net.IP,
	clock clock.Clock,
	registerNode bool,
	kubeClient clientset.Interface,
	nodeLabels map[string]string,
	nodeRef *v1.ObjectReference,
	nodeStatusUpdateFrequency time.Duration,
	nodeStatusReportFrequency time.Duration,
	nodeLister corelisters.NodeLister,
	providerID string,
	externalCloudProvider bool,
	cloud cloudprovider.Interface,
	registerSchedulable bool,
	registerWithTaints []api.Taint,
	enableControllerAttachDetach bool,
	keepTerminatedPodVolumes bool,
) Provider {
	return &nodeInfo{
		hostname:                     hostname,
		hostnameOverridden:           hostnameOverridden,
		nodeName:                     nodeName,
		nodeIP:                       nodeIP,
		clock:                        clock,
		registerNode:                 registerNode,
		kubeClient:                   kubeClient,
		nodeLabels:                   nodeLabels,
		nodeRef:                      nodeRef,
		nodeStatusUpdateFrequency:    nodeStatusUpdateFrequency,
		nodeStatusReportFrequency:    nodeStatusReportFrequency,
		nodeLister:                   nodeLister,
		providerID:                   providerID,
		externalCloudProvider:        externalCloudProvider,
		cloud:                        cloud,
		registerSchedulable:          registerSchedulable,
		registerWithTaints:           registerWithTaints,
		enableControllerAttachDetach: enableControllerAttachDetach,
		keepTerminatedPodVolumes:     keepTerminatedPodVolumes,
	}
}

// GetHostIP returns host IP or nil in case of error.
func (n *nodeInfo) GetHostIP() (net.IP, error) {
	node, err := n.GetNode()
	if err != nil {
		return nil, fmt.Errorf("cannot get node: %v", err)
	}
	return utilnode.GetNodeHostIP(node)
}

// GetHostIPAnyway attempts to return the host IP from kubelet's nodeInfo, or the initialNode.
func (n *nodeInfo) GetHostIPAnyWay() (net.IP, error) {
	node, err := n.GetNodeAnyWay()
	if err != nil {
		return nil, err
	}
	return utilnode.GetNodeHostIP(node)
}

// GetNodeAnyWay() must return a *v1.Node which is required by RunGeneralPredicates().
// The *v1.Node is obtained as follows:
// Return kubelet's nodeInfo for this node, except on error or if in standalone mode,
// in which case return a manufactured nodeInfo representing a node with no pods,
// zero capacity, and the default labels.
func (n *nodeInfo) GetNodeAnyWay() (*v1.Node, error) {
	if n.kubeClient != nil {
		if n, err := n.nodeLister.Get(string(n.nodeName)); err == nil {
			return n, nil
		}
	}
	return n.InitialNode(context.TODO())
}

// GetNode returns the node info for the configured node name of this Kubelet.
func (n *nodeInfo) GetNode() (*v1.Node, error) {
	if n.kubeClient == nil {
		return n.InitialNode(context.TODO())
	}
	return n.nodeLister.Get(string(n.nodeName))
}

// InitialNode constructs the initial v1.Node for this Kubelet, incorporating node
// labels, information from the cloud provider, and Kubelet configuration.
func (n *nodeInfo) InitialNode(ctx context.Context) (*v1.Node, error) {
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: string(n.nodeName),
			Labels: map[string]string{
				v1.LabelHostname:   n.hostname,
				v1.LabelOSStable:   runtime.GOOS,
				v1.LabelArchStable: runtime.GOARCH,
			},
		},
		Spec: v1.NodeSpec{
			Unschedulable: !n.registerSchedulable,
		},
	}
	osLabels, err := getOSSpecificLabels()
	if err != nil {
		return nil, err
	}
	for label, value := range osLabels {
		node.Labels[label] = value
	}

	nodeTaints := make([]v1.Taint, 0)
	if len(n.registerWithTaints) > 0 {
		taints := make([]v1.Taint, len(n.registerWithTaints))
		for i := range n.registerWithTaints {
			if err := k8s_api_v1.Convert_core_Taint_To_v1_Taint(&n.registerWithTaints[i], &taints[i], nil); err != nil {
				return nil, err
			}
		}
		nodeTaints = append(nodeTaints, taints...)
	}

	unschedulableTaint := v1.Taint{
		Key:    v1.TaintNodeUnschedulable,
		Effect: v1.TaintEffectNoSchedule,
	}

	// Taint node with TaintNodeUnschedulable when initializing
	// node to avoid race condition; refer to #63897 for more detail.
	if node.Spec.Unschedulable &&
		!taintutil.TaintExists(nodeTaints, &unschedulableTaint) {
		nodeTaints = append(nodeTaints, unschedulableTaint)
	}

	if n.externalCloudProvider {
		taint := v1.Taint{
			Key:    cloudproviderapi.TaintExternalCloudProvider,
			Value:  "true",
			Effect: v1.TaintEffectNoSchedule,
		}

		nodeTaints = append(nodeTaints, taint)
	}
	if len(nodeTaints) > 0 {
		node.Spec.Taints = nodeTaints
	}
	// Initially, set NodeNetworkUnavailable to true.
	if providerRequiresNetworkingConfiguration(n.cloud) {
		node.Status.Conditions = append(node.Status.Conditions, v1.NodeCondition{
			Type:               v1.NodeNetworkUnavailable,
			Status:             v1.ConditionTrue,
			Reason:             "NoRouteCreated",
			Message:            "Node created without a route",
			LastTransitionTime: metav1.NewTime(n.clock.Now()),
		})
	}

	if n.enableControllerAttachDetach {
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}

		klog.Infof("Setting node annotation to enable volume controller attach/detach")
		node.Annotations[volutil.ControllerManagedAttachAnnotation] = "true"
	} else {
		klog.Infof("Controller attach/detach is disabled for this node; Kubelet will attach and detach volumes")
	}

	if n.keepTerminatedPodVolumes {
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}
		klog.Infof("Setting node annotation to keep pod volumes of terminated pods attached to the node")
		node.Annotations[volutil.KeepTerminatedPodVolumesAnnotation] = "true"
	}

	// @question: should this be place after the call to the cloud provider? which also applies labels
	for k, v := range n.nodeLabels {
		if cv, found := node.ObjectMeta.Labels[k]; found {
			klog.Warningf("the node label %s=%s will overwrite default setting %s", k, v, cv)
		}
		node.ObjectMeta.Labels[k] = v
	}

	if n.providerID != "" {
		node.Spec.ProviderID = n.providerID
	}

	if n.cloud != nil {
		instances, ok := n.cloud.Instances()
		if !ok {
			return nil, fmt.Errorf("failed to get instances from cloud provider")
		}

		// TODO: We can't assume that the node has credentials to talk to the
		// cloudprovider from arbitrary nodes. At most, we should talk to a
		// local metadata server here.
		var err error
		if node.Spec.ProviderID == "" {
			node.Spec.ProviderID, err = cloudprovider.GetInstanceProviderID(ctx, n.cloud, n.nodeName)
			if err != nil {
				return nil, err
			}
		}

		instanceType, err := instances.InstanceType(ctx, n.nodeName)
		if err != nil {
			return nil, err
		}
		if instanceType != "" {
			klog.Infof("Adding node label from cloud provider: %s=%s", v1.LabelInstanceType, instanceType)
			node.ObjectMeta.Labels[v1.LabelInstanceType] = instanceType
			klog.Infof("Adding node label from cloud provider: %s=%s", v1.LabelInstanceTypeStable, instanceType)
			node.ObjectMeta.Labels[v1.LabelInstanceTypeStable] = instanceType
		}
		// If the cloud has zone information, label the node with the zone information
		zones, ok := n.cloud.Zones()
		if ok {
			zone, err := zones.GetZone(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get zone from cloud provider: %v", err)
			}
			if zone.FailureDomain != "" {
				klog.Infof("Adding node label from cloud provider: %s=%s", v1.LabelZoneFailureDomain, zone.FailureDomain)
				node.ObjectMeta.Labels[v1.LabelZoneFailureDomain] = zone.FailureDomain
				klog.Infof("Adding node label from cloud provider: %s=%s", v1.LabelZoneFailureDomainStable, zone.FailureDomain)
				node.ObjectMeta.Labels[v1.LabelZoneFailureDomainStable] = zone.FailureDomain
			}
			if zone.Region != "" {
				klog.Infof("Adding node label from cloud provider: %s=%s", v1.LabelZoneRegion, zone.Region)
				node.ObjectMeta.Labels[v1.LabelZoneRegion] = zone.Region
				klog.Infof("Adding node label from cloud provider: %s=%s", v1.LabelZoneRegionStable, zone.Region)
				node.ObjectMeta.Labels[v1.LabelZoneRegionStable] = zone.Region
			}
		}
	}

	n.SetNodeStatus(node)

	return node, nil
}

// setNodeStatus fills in the Status fields of the given Node, overwriting
// any fields that are currently set.
// TODO(madhusudancs): Simplify the logic for setting node conditions and
// refactor the node status condition code out to a different file.
func (n *nodeInfo) SetNodeStatus(node *v1.Node) {
	for i, f := range n.nodeStatusFuncs {
		klog.V(5).Infof("Setting node status at position %v", i)
		if err := f(node); err != nil {
			klog.Errorf("Failed to set some node status fields: %s", err)
		}
	}
}

// providerRequiresNetworkingConfiguration returns whether the cloud provider
// requires special networking configuration.
func providerRequiresNetworkingConfiguration(cloud cloudprovider.Interface) bool {
	// TODO: We should have a mechanism to say whether native cloud provider
	// is used or whether we are using overlay networking. We should return
	// true for cloud providers if they implement Routes() interface and
	// we are not using overlay networking.
	if cloud == nil || cloud.ProviderName() != "gce" {
		return false
	}
	_, supported := cloud.Routes()
	return supported
}
