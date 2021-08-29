// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAvailableLocalAddress(t *testing.T) {
	testEndpointAvailable(t, GetAvailableLocalAddress(t))
}

func TestGetAvailablePort(t *testing.T) {
	portStr := strconv.Itoa(int(GetAvailablePort(t)))
	require.NotEqual(t, "", portStr)

	testEndpointAvailable(t, "localhost:"+portStr)
}

func testEndpointAvailable(t *testing.T, endpoint string) {
	// Endpoint should be free.
	ln0, err := net.Listen("tcp", endpoint)
	require.NoError(t, err)
	require.NotNil(t, ln0)
	defer ln0.Close()

	// Ensure that the endpoint wasn't something like ":0" by checking that a
	// second listener will fail.
	ln1, err := net.Listen("tcp", endpoint)
	require.Error(t, err)
	require.Nil(t, ln1)
}

func TestCreateExclusionsList(t *testing.T) {
	// Test two examples of typical output from "netsh interface ipv4 show excludedportrange protocol=tcp"
	emptyExclusionsText := `

Protocol tcp Port Exclusion Ranges

Start Port    End Port      
----------    --------      

* - Administered port exclusions.`

	exclusionsText := `

Start Port    End Port
----------    --------
     49697       49796
     49797       49896

* - Administered port exclusions.
`
	exclusions := createExclusionsList(exclusionsText, t)
	require.Equal(t, len(exclusions), 2)

	emptyExclusions := createExclusionsList(emptyExclusionsText, t)
	require.Equal(t, len(emptyExclusions), 0)
}
