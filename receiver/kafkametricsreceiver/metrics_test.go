// Copyright  The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kafkametricsreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/collector/consumer/pdata"
)

func TestInitializeTopicMetrics(t *testing.T) {
	m := pdata.NewMetricSlice()
	tm := initializeTopicMetrics(&m)
	assert.NotNil(t, tm.partitions)
	assert.NotNil(t, tm.currentOffset)
	assert.NotNil(t, tm.oldestOffset)
	assert.NotNil(t, tm.replicas)
	assert.NotNil(t, tm.replicasInSync)

	assert.Equal(t, tm.partitions.Name(), partitionsName)
	assert.Equal(t, tm.currentOffset.Name(), currentOffsetName)
	assert.Equal(t, tm.oldestOffset.Name(), oldestOffsetName)
	assert.Equal(t, tm.replicasInSync.Name(), replicasInSyncName)
	assert.Equal(t, tm.replicas.Name(), replicasName)

	assert.Equal(t, tm.partitions.Description(), partitionsDescription)
	assert.Equal(t, tm.currentOffset.Description(), currentOffsetDescription)
	assert.Equal(t, tm.oldestOffset.Description(), oldestOffsetDescription)
	assert.Equal(t, tm.replicas.Description(), replicasDescription)
	assert.Equal(t, tm.replicasInSync.Description(), replicasInSyncDescription)
}

func TestInitializeBrokerMetrics(t *testing.T) {
	m := pdata.NewMetricSlice()
	bm := initializeBrokerMetrics(&m)
	assert.NotNil(t, bm.brokers)

	assert.Equal(t, bm.brokers.Name(), brokersName)
	assert.Equal(t, bm.brokers.Description(), brokersDescription)
}

func TestInitializeConsumerMetrics(t *testing.T) {
	m := pdata.NewMetricSlice()
	cm := initializeConsumerMetrics(&m)

	assert.NotNil(t, cm.offsetSum)
	assert.NotNil(t, cm.consumerLag)
	assert.NotNil(t, cm.consumerOffset)
	assert.NotNil(t, cm.groupMembers)
	assert.NotNil(t, cm.lagSum)
}

func TestAddPartitionsToMetric(t *testing.T) {
	ms := pdata.NewMetricSlice()
	tm := initializeTopicMetrics(&ms)
	m := tm.partitions
	assert.Equal(t, m.DataType(), pdata.MetricDataTypeIntGauge)
	topic := "test"
	var partitions int64 = 5
	m.IntGauge().DataPoints().Resize(1)
	addPartitionsToMetric(topic, partitions, m, 0)
	dp := m.IntGauge().DataPoints().At(0)
	assert.Equal(t, dp.Value(), partitions)
	sm := pdata.NewStringMap()
	sm.InitFromMap(map[string]string{
		"topic": topic,
	})
	assert.Equal(t, dp.LabelsMap(), sm)
}

func TestAddPartitionsDPToMetric(t *testing.T) {
	ms := pdata.NewMetricSlice()
	tm := initializeTopicMetrics(&ms)
	m := tm.partitions
	assert.Equal(t, m.DataType(), pdata.MetricDataTypeIntGauge)
	topic := "test"
	var partition int32 = 5
	var value int64 = 1
	addPartitionDPToMetric(topic, partition, value, m)
	dp := m.IntGauge().DataPoints().At(0)
	assert.Equal(t, dp.Value(), value)
	sm := pdata.NewStringMap()
	sm.InitFromMap(map[string]string{
		"topic":     topic,
		"partition": int32ToStr(partition),
	})
	assert.Equal(t, dp.LabelsMap(), sm)
}

func TestAddBrokersToMetric(t *testing.T) {
	ms := pdata.NewMetricSlice()
	bm := initializeBrokerMetrics(&ms)
	m := bm.brokers
	assert.Equal(t, m.DataType(), pdata.MetricDataTypeIntGauge)
	var brokers int64 = 5
	addBrokersToMetric(brokers, m)
	dp := m.IntGauge().DataPoints().At(0)
	assert.Equal(t, dp.Value(), brokers)
}

func TestAddGroupMembersToMetric(t *testing.T) {
	ms := pdata.NewMetricSlice()
	cm := initializeConsumerMetrics(&ms)
	m := cm.groupMembers
	assert.Equal(t, m.DataType(), pdata.MetricDataTypeIntGauge)
	var members int64 = 5
	group := "test"
	addGroupMembersToMetric(group, members, m)
	dp := m.IntGauge().DataPoints().At(0)
	assert.Equal(t, dp.Value(), members)
	sm := pdata.NewStringMap()
	sm.InitFromMap(map[string]string{"group": group})
	assert.Equal(t, dp.LabelsMap(), sm)
}

func TestAddConsumerPartitionDPToMetric(t *testing.T) {
	ms := pdata.NewMetricSlice()
	cm := initializeConsumerMetrics(&ms)
	m := cm.consumerLag
	assert.Equal(t, m.DataType(), pdata.MetricDataTypeIntGauge)
	group := "test_group"
	topic := "test_topic"
	var partition int32 = 1
	var value int64 = 1
	addConsumerPartitionDPToMetric(group, topic, partition, value, m)
	dp := m.IntGauge().DataPoints().At(0)
	assert.Equal(t, dp.Value(), value)
	sm := dp.LabelsMap()
	sm.InitFromMap(map[string]string{
		"topic":     topic,
		"group":     group,
		"partition": int32ToStr(partition),
	})
	assert.Equal(t, dp.LabelsMap(), sm)
}

func TestAddConsumerSumDPToMetric(t *testing.T) {
	ms := pdata.NewMetricSlice()
	cm := initializeConsumerMetrics(&ms)
	m := cm.lagSum
	assert.Equal(t, m.DataType(), pdata.MetricDataTypeIntGauge)
	group := "test_group"
	topic := "test_topic"
	var value int64 = 1
	addConsumerSumDPToMetric(group, topic, value, m)
	dp := m.IntGauge().DataPoints().At(0)
	assert.Equal(t, dp.Value(), value)
	sm := dp.LabelsMap()
	sm.InitFromMap(map[string]string{
		"topic": topic,
		"group": group,
	})
	assert.Equal(t, dp.LabelsMap(), sm)
}
