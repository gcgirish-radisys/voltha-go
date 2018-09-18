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
package core

import (
	"crypto/rand"
	"encoding/hex"
	m "math/rand"
)

func randomHex(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateDeviceId produces a device ID. DeviceId is 16 hex long - lower 12 hex is the device id.
// TODO:  A cluster unique ID may be required
func CreateDeviceId() string {
	val, _ := randomHex(12)
	return val
}

// CreateLogicalDeviceId is not used for now as the logical device ID is derived from the
// OLT MAC address
func CreateLogicalDeviceId() string {
	// logical device id is 16 hex long - lower 12 hex is the logical device id.  For now just generate the 12 hex
	val, _ := randomHex(12)
	return val
}

// CreateLogicalPortId produces a random port ID for a logical device.   
func CreateLogicalPortId() uint32 {
	//	A logical port is a uint32
	return m.Uint32()
}
