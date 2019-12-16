package model

import (
	"time"

	"github.com/signalfx/signalfx-agent/pkg/core/config"
	"github.com/vmware/govmomi/vim25/types"
)

// "real-time" vsphereInfo metrics are available in 20 second intervals
const VSMetricsInterval = 20

const (
	DatacenterType = "Datacenter"
	ClusterType    = "ClusterComputeResource"
	VMType         = "VirtualMachine"
	HostType       = "HostSystem"
)

// Config for the vSphere monitor
type Config struct {
	config.MonitorConfig `yaml:",inline" acceptsEndpoints:"true"`
	Host                 string `yaml:"host"`
	Port                 uint16 `yaml:"port"`
	// The vSphere username
	Username string `yaml:"username"`
	// The vSphere password
	Password string `yaml:"password"`
	// Whether we verify the server's certificate chain and host name
	InsecureSkipVerify bool `yaml:"insecureSkipVerify"`
	// How often to reload the inventory and inventory metrics
	InventoryRefreshInterval time.Duration `yaml:"inventoryRefreshInterval" default:"600s"`

	// Path to the ca file
	TLSCACertPath string `yaml:"tlsCACertPath"`

	// Configure client certs. Both tlsClientKeyPath and tlsClientCertificatePath must be present. The files must contain
	// PEM encoded data.
	// Path to the client certificate
	TLSClientCertificatePath string `yaml:"tlsClientCertificatePath"`
	// Path to the keyfile
	TLSClientKeyPath string `yaml:"tlsClientKeyPath"`
}

type dimensions map[string]string

type InventoryObject struct {
	Ref        types.ManagedObjectReference
	MetricIds  []types.PerfMetricId
	dimensions dimensions
}

type Inventory struct {
	Objects      []*InventoryObject
	DimensionMap map[string]dimensions
}

func NewInventoryObject(ref types.ManagedObjectReference, extraDimensions map[string]string) *InventoryObject {
	dimensions := map[string]string{
		"ref_id":      ref.Value,
		"object_type": ref.Type,
	}
	for key, value := range extraDimensions {
		dimensions[key] = value
	}
	return &InventoryObject{
		Ref:        ref,
		dimensions: dimensions,
	}
}

func NewInventory() *Inventory {
	inv := &Inventory{}
	inv.DimensionMap = make(map[string]dimensions)
	return inv
}

func (inv *Inventory) AddObject(obj *InventoryObject) {
	inv.Objects = append(inv.Objects, obj)
	inv.DimensionMap[obj.Ref.Value] = obj.dimensions
}

type MetricInfosByKey map[int32]MetricInfo

type MetricInfo struct {
	MetricName      string
	PerfCounterInfo types.PerfCounterInfo
}

type VsphereInfo struct {
	Inv              *Inventory
	PerfCounterIndex MetricInfosByKey
}
