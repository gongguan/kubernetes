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
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	utilnode "k8s.io/kubernetes/pkg/util/node"
	"net"
)

var _ Provider = &nodeInfo{}

// Provider defines the interface of kubelet node information.
type Provider interface {
	GetHostIP() (net.IP, error)
	GetHostIPAnyWay() (net.IP, error)
	GetNodeAnyWay() (*v1.Node, error)
	GetNode() (*v1.Node, error)
	SetInitialNodeFunc(fn func() (*v1.Node, error))
	GetHostname() string
	GetHostnameOverridden() bool
	GetNodeName() types.NodeName
	GetNodeIP() net.IP
	GetNodeRef() *v1.ObjectReference
	GetNodeLabels() map[string]string
	SetNodeLister(lister corelisters.NodeLister)
}

type nodeInfo struct {
	// hostname is the hostname the kubelet detected or was given via flag/config
	hostname string

	// hostnameOverridden indicates the hostname was overridden via flag/config
	hostnameOverridden bool

	nodeName types.NodeName

	// If non-nil, use this IP address for the node
	nodeIP net.IP

	kubeClient clientset.Interface

	// a list of node labels to register
	nodeLabels map[string]string

	// Reference to this node.
	nodeRef *v1.ObjectReference

	nodeLister corelisters.NodeLister

	initialNodeFunc func() (*v1.Node, error)
}

// New creates nodeInfo Provider which provider node information.
func New(
	hostname string,
	hostnameOverridden bool,
	nodeName types.NodeName,
	nodeIP net.IP,
	kubeClient clientset.Interface,
	nodeLabels map[string]string,
	nodeRef *v1.ObjectReference,
	nodeLister corelisters.NodeLister,
) Provider {
	return &nodeInfo{
		hostname:           hostname,
		hostnameOverridden: hostnameOverridden,
		nodeName:           nodeName,
		nodeIP:             nodeIP,
		kubeClient:         kubeClient,
		nodeLabels:         nodeLabels,
		nodeRef:            nodeRef,
		nodeLister:         nodeLister,
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
	return n.initialNodeFunc()
}

// GetNode returns the node info for the configured node name of this Kubelet.
func (n *nodeInfo) GetNode() (*v1.Node, error) {
	if n.kubeClient == nil {
		return n.initialNodeFunc()
	}
	return n.nodeLister.Get(string(n.nodeName))
}

func (n *nodeInfo) SetInitialNodeFunc(fn func() (*v1.Node, error)) {
	n.initialNodeFunc = fn
}

// GetHostname Returns the nodeInfo hostname.
func (n *nodeInfo) GetHostname() string {
	return n.hostname
}

// GetHostnameOverridden Returns the hostnameOverridden.
func (n *nodeInfo) GetHostnameOverridden() bool {
	return n.hostnameOverridden
}

// GetHostname Returns the nodeInfo nodeName.
func (n *nodeInfo) GetNodeName() types.NodeName {
	return n.nodeName
}

// GetNodeIP Returns the nodeInfo nodeIP.
func (n *nodeInfo) GetNodeIP() net.IP {
	return n.nodeIP
}

// GetNodeRef Returns the nodeInfo nodeRef.
func (n *nodeInfo) GetNodeRef() *v1.ObjectReference {
	return n.nodeRef
}

// GetNodeLabels Returns the nodeInfo nodeLabels
func (n *nodeInfo) GetNodeLabels() map[string]string {
	return n.nodeLabels
}

// SetNodeLister Sets the nodeInfo nodeLister.
// TODO SetNodeLister only used for test, remove it after refactor tests.
func (n *nodeInfo) SetNodeLister(lister corelisters.NodeLister) {
	n.nodeLister = lister
}
