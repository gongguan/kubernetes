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
	"time"

	"github.com/stretchr/testify/mock"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	clientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/cloud-provider"

	api "k8s.io/kubernetes/pkg/apis/core"
	utilnode "k8s.io/kubernetes/pkg/util/node"
)

var _ Provider = &FakeNodeInfo{}

// FakeNodeInfo mock nodeinfo.Provider implementation.
type FakeNodeInfo struct {
	mock.Mock
	Hostname                     string
	HostnameOverridden           bool
	NodeName                     types.NodeName
	NodeIP                       net.IP
	Clock                        clock.Clock
	RegisterNode                 bool
	RegistrationCompleted        bool
	KubeClient                   clientset.Interface
	NodeLabels                   map[string]string
	NodeRef                      *v1.ObjectReference
	NodeLister                   corelisters.NodeLister
	ProviderID                   string
	ExternalCloudProvider        bool
	Cloud                        cloudprovider.Interface
	RegisterSchedulable          bool
	RegisterWithTaints           []api.Taint
	EnableControllerAttachDetach bool
	NodeStatusUpdateFrequency    time.Duration
	NodeStatusReportFrequency    time.Duration
	LastStatusReportTime         time.Time
	NodeStatusFuncs              []func(*v1.Node) error
	KeepTerminatedPodVolumes     bool
}

// GetHostIP is a mock implementation of Provider.GetHostIP.
func (fn *FakeNodeInfo) GetHostIP() (net.IP, error) {
	node, err := fn.GetNode()
	if err != nil {
		return nil, fmt.Errorf("cannot get node: %v", err)
	}
	return utilnode.GetNodeHostIP(node)
}

// GetHostIPAnyWay is a mock implementation of Provider.GetHostIPAnyWay.
func (fn *FakeNodeInfo) GetHostIPAnyWay() (net.IP, error) {
	return fn.GetHostIP()
}

// GetNodeAnyWay is a mock implementation of Provider.GetNodeAnyWay.
func (fn *FakeNodeInfo) GetNodeAnyWay() (*v1.Node, error) {
	return fn.InitialNode(context.TODO())
}

// GetNode is a mock implementation of Provider.GetNode.
func (fn *FakeNodeInfo) GetNode() (*v1.Node, error) {
	return fn.InitialNode(context.TODO())
}

// InitialNode is a mock implementation of Provider.InitialNode.
func (fn *FakeNodeInfo) InitialNode(ctx context.Context) (*v1.Node, error) {
	ret := fn.Called()
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

// SetNodeStatus is a mock implementation of Provider.SetNodeStatus.
func (fn *FakeNodeInfo) SetNodeStatus(node *v1.Node) {}

// GetHostname is a mock implementation of Provider.GetHostname.
func (fn *FakeNodeInfo) GetHostname() string {
	return fn.Hostname
}

// GetHostnameOverridden is a mock implementation of Provider.GetHostnameOverridden.
func (fn *FakeNodeInfo) GetHostnameOverridden() bool {
	return fn.HostnameOverridden
}

// GetNodeName is a mock implementation of Provider.GetNodeName.
func (fn *FakeNodeInfo) GetNodeName() types.NodeName {
	return fn.NodeName
}

// GetNodeIP is a mock implementation of Provider.GetNodeIP.
func (fn *FakeNodeInfo) GetNodeIP() net.IP {
	return fn.NodeIP
}

// GetRegisterNode is a mock implementation of Provider.GetRegisterNode.
func (fn *FakeNodeInfo) GetRegisterNode() bool {
	return fn.RegisterNode
}

// GetRegistrationCompleted is a mock implementation of Provider.GetRegistrationCompleted.
func (fn *FakeNodeInfo) GetRegistrationCompleted() bool {
	return fn.RegistrationCompleted
}

// GetNodeRef is a mock implementation of Provider.GetNodeRef.
func (fn *FakeNodeInfo) GetNodeRef() *v1.ObjectReference {
	return fn.NodeRef
}

// GetNodeStatusUpdateFrequency is a mock implementation of Provider.GetNodeStatusUpdateFrequency.
func (fn *FakeNodeInfo) GetNodeStatusUpdateFrequency() time.Duration {
	return fn.NodeStatusUpdateFrequency
}

// GetNodeStatusReportFrequency is a mock implementation of Provider.GetNodeStatusReportFrequency.
func (fn *FakeNodeInfo) GetNodeStatusReportFrequency() time.Duration {
	return fn.NodeStatusReportFrequency
}

// GetLastStatusReportTime is a mock implementation of Provider.GetLastStatusReportTime.
func (fn *FakeNodeInfo) GetLastStatusReportTime() time.Time {
	return fn.LastStatusReportTime
}

// SetRegistrationCompleted is a mock implementation of Provider.SetRegistrationCompleted.
func (fn *FakeNodeInfo) SetRegistrationCompleted(value bool) {}

// SetNodeStatusReportFrequency is a mock implementation of Provider.SetNodeStatusReportFrequency.
func (fn *FakeNodeInfo) SetNodeStatusReportFrequency(duration time.Duration) {}

// SetLastStatusReportTime is a mock implementation of Provider.SetLastStatusReportTime.
func (fn *FakeNodeInfo) SetLastStatusReportTime(time time.Time) {}

// SetNodeLister is a mock implementation of Provider.SetNodeLister.
func (fn *FakeNodeInfo) SetNodeLister(lister corelisters.NodeLister) {}

// SetRegisterSchedulable is a mock implementation of Provider.SetRegisterSchedulable.
func (fn *FakeNodeInfo) SetRegisterSchedulable(schedulable bool) {}

// SetNodeStatusFuncs is a mock implementation of Provider.SetNodeStatusFuncs.
func (fn *FakeNodeInfo) SetNodeStatusFuncs(funcs []func(*v1.Node) error) {}
