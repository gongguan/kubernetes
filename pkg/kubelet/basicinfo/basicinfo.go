package basicinfo

import (
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"k8s.io/kubernetes/pkg/kubelet/config"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilnode "k8s.io/kubernetes/pkg/util/node"
	corelisters "k8s.io/client-go/listers/core/v1"
	"net"
	"context"
	"k8s.io/cloud-provider"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog"
	"fmt"
	"runtime"
	kubeletapis "k8s.io/kubernetes/pkg/kubelet/apis"
	k8s_api_v1 "k8s.io/kubernetes/pkg/apis/core/v1"
	taintutil "k8s.io/kubernetes/pkg/util/taints"
	cloudproviderapi "k8s.io/cloud-provider/api"
	volutil "k8s.io/kubernetes/pkg/volume/util"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/apimachinery/pkg/util/clock"
)

const (
	EtcHostsPath                      = "/etc/hosts"
	ManagedHostsHeader                = "# Kubernetes-managed hosts file.\n"
	ManagedHostsHeaderWithHostNetwork = "# Kubernetes-managed hosts file (host network).\n"
)

type serviceLister interface {
	List(labels.Selector) ([]*v1.Service, error)
}

type BasicInfo struct {
	// hostname is the hostname the kubelet detected or was given via flag/config
	hostname string

	nodeName types.NodeName

	// clock is an interface that provides time related functionality in a way that makes it
	// easy to test the code.
	clock clock.Clock

	rootDirectory   string

	KubeClient clientset.Interface

	MasterServiceNamespace string

	// a list of node labels to register
	nodeLabels map[string]string

	// The EventRecorder to use
	Recorder record.EventRecorder
	// serviceLister knows how to list services
	ServiceLister serviceLister

	nodeLister corelisters.NodeLister

	// If non-nil, this is a unique identifier for the node in an external database, eg. cloudprovider
	providerID string

	// Indicates that the node initialization happens in an external cloud controller
	externalCloudProvider bool

	// Cloud provider interface.
	cloud cloudprovider.Interface

	// Set to true to have the node register itself as schedulable.
	registerSchedulable bool

	// List of taints to add to a node object when the kubelet registers itself.
	registerWithTaints []api.Taint

	// enableControllerAttachDetach indicates the Attach/Detach controller
	// should manage attachment/detachment of volumes scheduled to this node,
	// and disable kubelet from executing any attach/detach operations
	enableControllerAttachDetach bool

	ExperimentalHostUserNamespaceDefaulting bool

	// handlers called during the tryUpdateNodeStatus cycle
	NodeStatusFuncs []func(*v1.Node) error

	// This flag, if set, instructs the kubelet to keep volumes from terminated pods mounted to the node.
	// This can be useful for debugging volume related issues.
	keepTerminatedPodVolumes bool // DEPRECATED
}

func NewBasicInfo(
	hostname string,
	nodeName types.NodeName,
	clock clock.Clock,
	rootDirectory string,
	kubeClient clientset.Interface,
	masterServiceNamespace string,
	nodeLabels map[string]string,
	recorder record.EventRecorder,
	serviceLister serviceLister,
	nodeLister corelisters.NodeLister,
	providerID string,
	externalCloudProvider bool,
	cloud cloudprovider.Interface,
	registerSchedulable bool,
	registerWithTaints []api.Taint,
	enableControllerAttachDetach bool,
	experimentalHostUserNamespaceDefaulting bool,
	keepTerminatedPodVolumes bool,
) *BasicInfo {
	return &BasicInfo{
		hostname: hostname,
		nodeName: nodeName,
		clock:    clock,
		rootDirectory: rootDirectory,
		KubeClient: kubeClient,
		MasterServiceNamespace: masterServiceNamespace,
		nodeLabels: nodeLabels,
		Recorder: recorder,
		ServiceLister: serviceLister,
		nodeLister: nodeLister,
		providerID: providerID,
		externalCloudProvider: externalCloudProvider,
		cloud: cloud,
		registerSchedulable: registerSchedulable,
		registerWithTaints: registerWithTaints,
		enableControllerAttachDetach: enableControllerAttachDetach,
		ExperimentalHostUserNamespaceDefaulting: experimentalHostUserNamespaceDefaulting,
		keepTerminatedPodVolumes: keepTerminatedPodVolumes,
	}
}

func (b *BasicInfo) GetPodDir(podUID types.UID) string {
	return filepath.Join(b.getPodsDir(), string(podUID))
}

func (b *BasicInfo) getPodsDir() string {
	return filepath.Join(b.getRootDir(), config.DefaultKubeletPodsDirName)
}

func (b *BasicInfo) getRootDir() string {
	return b.rootDirectory
}

func (b *BasicInfo) GetPodContainerDir(podUID types.UID, ctrName string) string {
	return filepath.Join(b.GetPodDir(podUID), config.DefaultKubeletContainersDirName, ctrName)
}

// getHostIPAnyway attempts to return the host IP from kubelet's nodeInfo, or
// the initialNode.
func (b *BasicInfo) GetHostIPAnyWay() (net.IP, error) {
	node, err := b.GetNodeAnyWay()
	if err != nil {
		return nil, err
	}
	return utilnode.GetNodeHostIP(node)
}

// getNodeAnyWay() must return a *v1.Node which is required by RunGeneralPredicates().
// The *v1.Node is obtained as follows:
// Return kubelet's nodeInfo for this node, except on error or if in standalone mode,
// in which case return a manufactured nodeInfo representing a node with no pods,
// zero capacity, and the default labels.
func (b *BasicInfo) GetNodeAnyWay() (*v1.Node, error) {
	if b.KubeClient != nil {
		if n, err := b.nodeLister.Get(string(b.nodeName)); err == nil {
			return n, nil
		}
	}
	return b.initialNode(context.TODO())
}

// GetNode returns the node info for the configured node name of this Kubelet.
func (b *BasicInfo) GetNode() (*v1.Node, error) {
	if b.KubeClient == nil {
		return b.initialNode(context.TODO())
	}
	return b.nodeLister.Get(string(b.nodeName))
}

// initialNode constructs the initial v1.Node for this Kubelet, incorporating node
// labels, information from the cloud provider, and Kubelet configuration.
func (b *BasicInfo) initialNode(ctx context.Context) (*v1.Node, error) {
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: string(b.nodeName),
			Labels: map[string]string{
				v1.LabelHostname:      b.hostname,
				v1.LabelOSStable:      runtime.GOOS,
				v1.LabelArchStable:    runtime.GOARCH,
				kubeletapis.LabelOS:   runtime.GOOS,
				kubeletapis.LabelArch: runtime.GOARCH,
			},
		},
		Spec: v1.NodeSpec{
			Unschedulable: !b.registerSchedulable,
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
	if len(b.registerWithTaints) > 0 {
		taints := make([]v1.Taint, len(b.registerWithTaints))
		for i := range b.registerWithTaints {
			if err := k8s_api_v1.Convert_core_Taint_To_v1_Taint(&b.registerWithTaints[i], &taints[i], nil); err != nil {
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

	if b.externalCloudProvider {
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
	if b.providerRequiresNetworkingConfiguration() {
		node.Status.Conditions = append(node.Status.Conditions, v1.NodeCondition{
			Type:               v1.NodeNetworkUnavailable,
			Status:             v1.ConditionTrue,
			Reason:             "NoRouteCreated",
			Message:            "Node created without a route",
			LastTransitionTime: metav1.NewTime(b.clock.Now()),
		})
	}

	if b.enableControllerAttachDetach {
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}

		klog.Infof("Setting node annotation to enable volume controller attach/detach")
		node.Annotations[volutil.ControllerManagedAttachAnnotation] = "true"
	} else {
		klog.Infof("Controller attach/detach is disabled for this node; Kubelet will attach and detach volumes")
	}

	if b.keepTerminatedPodVolumes {
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}
		klog.Infof("Setting node annotation to keep pod volumes of terminated pods attached to the node")
		node.Annotations[volutil.KeepTerminatedPodVolumesAnnotation] = "true"
	}

	// @question: should this be place after the call to the cloud provider? which also applies labels
	for k, v := range b.nodeLabels {
		if cv, found := node.ObjectMeta.Labels[k]; found {
			klog.Warningf("the node label %s=%s will overwrite default setting %s", k, v, cv)
		}
		node.ObjectMeta.Labels[k] = v
	}

	if b.providerID != "" {
		node.Spec.ProviderID = b.providerID
	}

	if b.cloud != nil {
		instances, ok := b.cloud.Instances()
		if !ok {
			return nil, fmt.Errorf("failed to get instances from cloud provider")
		}

		// TODO: We can't assume that the node has credentials to talk to the
		// cloudprovider from arbitrary nodes. At most, we should talk to a
		// local metadata server here.
		var err error
		if node.Spec.ProviderID == "" {
			node.Spec.ProviderID, err = cloudprovider.GetInstanceProviderID(ctx, b.cloud, b.nodeName)
			if err != nil {
				return nil, err
			}
		}

		instanceType, err := instances.InstanceType(ctx, b.nodeName)
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
		zones, ok := b.cloud.Zones()
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

	b.setNodeStatus(node)

	return node, nil
}

// setNodeStatus fills in the Status fields of the given Node, overwriting
// any fields that are currently set.
// TODO(madhusudancs): Simplify the logic for setting node conditions and
// refactor the node status condition code out to a different file.
func (b *BasicInfo) setNodeStatus(node *v1.Node) {
	for i, f := range b.NodeStatusFuncs {
		klog.V(5).Infof("Setting node status at position %v", i)
		if err := f(node); err != nil {
			klog.Errorf("Failed to set some node status fields: %s", err)
		}
	}
}

// providerRequiresNetworkingConfiguration returns whether the cloud provider
// requires special networking configuration.
func (b *BasicInfo) providerRequiresNetworkingConfiguration() bool {
	// TODO: We should have a mechanism to say whether native cloud provider
	// is used or whether we are using overlay networking. We should return
	// true for cloud providers if they implement Routes() interface and
	// we are not using overlay networking.
	if b.cloud == nil || b.cloud.ProviderName() != "gce" {
		return false
	}
	_, supported := b.cloud.Routes()
	return supported
}
