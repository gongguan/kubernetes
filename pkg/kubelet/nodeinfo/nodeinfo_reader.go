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
	"net"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

type nodeInfoReader interface {
	GetHostname() string
	GetHostnameOverridden() bool
	GetNodeName() types.NodeName
	GetNodeIP() net.IP
	GetRegisterNode() bool
	GetRegistrationCompleted() bool
	GetNodeRef() *v1.ObjectReference
	GetNodeStatusUpdateFrequency() time.Duration
	GetNodeStatusReportFrequency() time.Duration
	GetLastStatusReportTime() time.Time
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

// GetRegisterNode Returns the nodeInfo registerNode.
func (n *nodeInfo) GetRegisterNode() bool {
	return n.registerNode
}

// GetRegistrationCompleted Returns the nodeInfo registrationCompleted.
func (n *nodeInfo) GetRegistrationCompleted() bool {
	return n.registrationCompleted
}

// GetNodeRef Returns the nodeInfo nodeRef.
func (n *nodeInfo) GetNodeRef() *v1.ObjectReference {
	return n.nodeRef
}

// GetNodeStatusUpdateFrequency Returns the nodeInfo nodeStatusUpdateFrequency.
func (n *nodeInfo) GetNodeStatusUpdateFrequency() time.Duration {
	return n.nodeStatusUpdateFrequency
}

// GetNodeStatusReportFrequency Returns the nodeInfo nodeStatusReportFrequency.
func (n *nodeInfo) GetNodeStatusReportFrequency() time.Duration {
	return n.nodeStatusReportFrequency
}

// GetLastStatusReportTime Returns the nodeInfo lastStatusReportTime.
func (n *nodeInfo) GetLastStatusReportTime() time.Time {
	return n.lastStatusReportTime
}
