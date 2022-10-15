/*
Copyright 2022.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CCPSpec defines the desired state of CCP
type CCPSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of CCP. Edit ccp_types.go to remove/update
	ConnectionProfile ConnectionProfile `json:"connectionProfile"`

	// +kubebuilder:default:=30
	SyncTime int64 `json:"syncTime,omitempty"`
}
type ConnectionProfile struct {
	Name          string           `json:"name"`
	Version       string           `json:"version"`
	Client        Client           `json:"client"`
	Organizations map[string]MSP   `json:"organizations"`
	Peers         map[string]Peer  `json:"peers"`
	Channels      map[string]Peers `json:"channels"`
}
type Client struct {
	Organization string `json:"organization"`
}
type MSP struct {
	Mspid string              `json:"mspid"`
	Peers []string            `json:"peers"`
	Users map[string]Identity `json:"users"`
}

type Peer struct {
	URL        string     `json:"url"`
	TLSCACerts TLSCACerts `json:"tlsCACerts"`
}
type TLSCACerts struct {
	Pem string `json:"pem"`
}

type Options struct {
	Path string
}

type Cert struct {
	Pem string `json:"pem"`
}
type Key struct {
	Pem string `json:"pem"`
}
type Identity struct {
	Cert Cert `json:"cert"`
	Key  Key  `json:"key"`
}

type Peers2 struct {
	ChaincodeQuery bool `json:"chaincodeQuery"`
	Discover       bool `json:"discover"`
	EndorsingPeer  bool `json:"endorsingPeer"`
	EventSource    bool `json:"eventSource"`
	LedgerQuery    bool `json:"ledgerQuery"`
}
type Peers struct {
	Peers map[string]Peers2 `json:"peers"`
}

// CCPStatus defines the observed state of CCP
type CCPStatus struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// CCP is the Schema for the ccps API
type CCP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CCPSpec   `json:"spec,omitempty"`
	Status CCPStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CCPList contains a list of CCP
type CCPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CCP `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CCP{}, &CCPList{})
}
