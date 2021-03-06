/*
 * Copyright 2018-present Open Networking Foundation

 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

 * http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package adaptercore

import (
	"context"
	"fmt"
	"github.com/gogo/protobuf/proto"
	com "github.com/opencord/voltha-go/adapters/common"
	"github.com/opencord/voltha-go/common/log"
	ic "github.com/opencord/voltha-protos/go/inter_container"
	of "github.com/opencord/voltha-protos/go/openflow_13"
	"github.com/opencord/voltha-protos/go/voltha"
	"strconv"
	"strings"
	"sync"
)

//DeviceHandler follows the same patterns as ponsim_olt.  The only difference is that it does not
// interact with an OLT device.
type DeviceHandler struct {
	deviceId     string
	deviceType   string
	device       *voltha.Device
	coreProxy    *com.CoreProxy
	simulatedOLT *SimulatedONU
	uniPort      *voltha.Port
	ponPort      *voltha.Port
	exitChannel  chan int
	lockDevice   sync.RWMutex
}

//NewDeviceHandler creates a new device handler
func NewDeviceHandler(cp *com.CoreProxy, device *voltha.Device, adapter *SimulatedONU) *DeviceHandler {
	var dh DeviceHandler
	dh.coreProxy = cp
	cloned := (proto.Clone(device)).(*voltha.Device)
	dh.deviceId = cloned.Id
	dh.deviceType = cloned.Type
	dh.device = cloned
	dh.simulatedOLT = adapter
	dh.exitChannel = make(chan int, 1)
	dh.lockDevice = sync.RWMutex{}
	return &dh
}

// start save the device to the data model
func (dh *DeviceHandler) start(ctx context.Context) {
	dh.lockDevice.Lock()
	defer dh.lockDevice.Unlock()
	log.Debugw("starting-device-agent", log.Fields{"device": dh.device})
	// Add the initial device to the local model
	log.Debug("device-agent-started")
}

// stop stops the device dh.  Not much to do for now
func (dh *DeviceHandler) stop(ctx context.Context) {
	dh.lockDevice.Lock()
	defer dh.lockDevice.Unlock()
	log.Debug("stopping-device-agent")
	dh.exitChannel <- 1
	log.Debug("device-agent-stopped")
}

func macAddressToUint32Array(mac string) []uint32 {
	slist := strings.Split(mac, ":")
	result := make([]uint32, len(slist))
	var err error
	var tmp int64
	for index, val := range slist {
		if tmp, err = strconv.ParseInt(val, 16, 32); err != nil {
			return []uint32{1, 2, 3, 4, 5, 6}
		}
		result[index] = uint32(tmp)
	}
	return result
}

func (dh *DeviceHandler) AdoptDevice(device *voltha.Device) {
	log.Debugw("AdoptDevice", log.Fields{"deviceId": device.Id})

	//	Update the device info
	cloned := proto.Clone(device).(*voltha.Device)
	cloned.Root = false
	cloned.Vendor = "simulators"
	cloned.Model = "go-simulators"
	//cloned.SerialNumber = com.GetRandomSerialNumber()
	cloned.MacAddress = strings.ToUpper(com.GetRandomMacAddress())

	// Synchronous call to update device - this method is run in its own go routine
	if err := dh.coreProxy.DeviceUpdate(nil, cloned); err != nil {
		log.Errorw("error-updating-device", log.Fields{"deviceId": device.Id, "error": err})
	}

	// Use the channel Id, assigned by the parent device to me, as the port number
	uni_port := uint32(2)
	if device.ProxyAddress != nil {
		if device.ProxyAddress.ChannelId != 0 {
			uni_port = device.ProxyAddress.ChannelId
		}
	}

	//	Now create the UNI Port
	dh.uniPort = &voltha.Port{
		PortNo:     uni_port,
		Label:      fmt.Sprintf("uni-%d", uni_port),
		Type:       voltha.Port_ETHERNET_UNI,
		AdminState: voltha.AdminState_ENABLED,
		OperStatus: voltha.OperStatus_ACTIVE,
	}

	// Synchronous call to update device - this method is run in its own go routine
	if err := dh.coreProxy.PortCreated(nil, cloned.Id, dh.uniPort); err != nil {
		log.Errorw("error-creating-nni-port", log.Fields{"deviceId": device.Id, "error": err})
	}

	//	Now create the PON Port
	dh.ponPort = &voltha.Port{
		PortNo:     1,
		Label:      fmt.Sprintf("pon-%d", 1),
		Type:       voltha.Port_PON_ONU,
		AdminState: voltha.AdminState_ENABLED,
		OperStatus: voltha.OperStatus_ACTIVE,
		Peers: []*voltha.Port_PeerPort{{DeviceId: cloned.ParentId,
			PortNo: cloned.ParentPortNo}},
	}

	// Synchronous call to update device - this method is run in its own go routine
	if err := dh.coreProxy.PortCreated(nil, cloned.Id, dh.ponPort); err != nil {
		log.Errorw("error-creating-nni-port", log.Fields{"deviceId": device.Id, "error": err})
	}

	cloned.ConnectStatus = voltha.ConnectStatus_REACHABLE
	cloned.OperStatus = voltha.OperStatus_ACTIVE

	//	Update the device state
	if err := dh.coreProxy.DeviceStateUpdate(nil, cloned.Id, cloned.ConnectStatus, cloned.OperStatus); err != nil {
		log.Errorw("error-creating-nni-port", log.Fields{"deviceId": device.Id, "error": err})
	}

	dh.device = cloned
}

func (dh *DeviceHandler) GetOfpPortInfo(device *voltha.Device, portNo int64) (*ic.PortCapability, error) {
	cap := uint32(of.OfpPortFeatures_OFPPF_1GB_FD | of.OfpPortFeatures_OFPPF_FIBER)
	return &ic.PortCapability{
		Port: &voltha.LogicalPort{
			OfpPort: &of.OfpPort{
				HwAddr:     macAddressToUint32Array(dh.device.MacAddress),
				Config:     0,
				State:      uint32(of.OfpPortState_OFPPS_LIVE),
				Curr:       cap,
				Advertised: cap,
				Peer:       cap,
				CurrSpeed:  uint32(of.OfpPortFeatures_OFPPF_1GB_FD),
				MaxSpeed:   uint32(of.OfpPortFeatures_OFPPF_1GB_FD),
			},
			DeviceId:     dh.device.Id,
			DevicePortNo: uint32(portNo),
		},
	}, nil
}

func (dh *DeviceHandler) Process_inter_adapter_message(msg *ic.InterAdapterMessage) error {
	log.Debugw("Process_inter_adapter_message", log.Fields{"msgId": msg.Header.Id})
	return nil
}
