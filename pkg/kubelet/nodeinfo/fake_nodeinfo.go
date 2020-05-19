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
	"github.com/stretchr/testify/mock"
	"net"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	utilnode "k8s.io/kubernetes/pkg/util/node"
)

var _ Provider = &FakeNodeInfo{}

// FakeNodeInfo mock nodeinfo.Provider implementation.
type FakeNodeInfo struct {
	mock.Mock
	Hostname           string
	HostnameOverridden bool
	NodeName           types.NodeName
	NodeIP             net.IP
	KubeClient         clientset.Interface
	NodeLabels         map[string]string
	NodeRef            *v1.ObjectReference
	NodeLister         corelisters.NodeLister
	InitialNodeFunc    func() (*v1.Node, error)
}

// GetHostIP is a mock implementation of Provider.GetHostIP.
func (fni *FakeNodeInfo) GetHostIP() (net.IP, error) {
	node, err := fni.GetNode()
	if err != nil {
		return nil, fmt.Errorf("cannot get node: %v", err)
	}
	return utilnode.GetNodeHostIP(node)
}

// GetHostIPAnyWay is a mock implementation of Provider.GetHostIPAnyWay.
func (fni *FakeNodeInfo) GetHostIPAnyWay() (net.IP, error) {
	return fni.GetHostIP()
}

// GetNodeAnyWay is a mock implementation of Provider.GetNodeAnyWay.
func (fni *FakeNodeInfo) GetNodeAnyWay() (*v1.Node, error) {
	return fni.initialNode(context.TODO())
}

// GetNode is a mock implementation of Provider.GetNode.
func (fni *FakeNodeInfo) GetNode() (*v1.Node, error) {
	return fni.initialNode(context.TODO())
}

func (fni *FakeNodeInfo) SetInitialNodeFunc(fn func() (*v1.Node, error)) {
	fni.InitialNodeFunc = fn
}

// InitialNode is a mock implementation of Provider.InitialNode.
func (fni *FakeNodeInfo) initialNode(ctx context.Context) (*v1.Node, error) {
	ret := fni.Called()
	var r0 *v1.Node
	if rf, ok := ret.Get(0).(func() *v1.Node); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1.Node)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}
	return r0, r1
}

// GetHostname is a mock implementation of Provider.GetHostname.
func (fni *FakeNodeInfo) GetHostname() string {
	return fni.Hostname
}

// GetHostnameOverridden is a mock implementation of Provider.GetHostnameOverridden.
func (fni *FakeNodeInfo) GetHostnameOverridden() bool {
	return fni.HostnameOverridden
}

// GetNodeName is a mock implementation of Provider.GetNodeName.
func (fni *FakeNodeInfo) GetNodeName() types.NodeName {
	return fni.NodeName
}

// GetNodeIP is a mock implementation of Provider.GetNodeIP.
func (fni *FakeNodeInfo) GetNodeIP() net.IP {
	return fni.NodeIP
}

// GetNodeRef is a mock implementation of Provider.GetNodeRef.
func (fni *FakeNodeInfo) GetNodeRef() *v1.ObjectReference {
	return fni.NodeRef
}

// GetNodeLabels is a mock implementation of Provider.GetNodeLabels.
func (fni *FakeNodeInfo) GetNodeLabels() map[string]string {
	return fni.NodeLabels
}

// SetNodeLister is a mock implementation of Provider.SetNodeLister.
func (fni *FakeNodeInfo) SetNodeLister(lister corelisters.NodeLister) {}
