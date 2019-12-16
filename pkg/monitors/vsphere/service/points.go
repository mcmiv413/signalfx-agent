package service

import (
	"time"

	"github.com/signalfx/golib/v3/datapoint"
	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/vim25/types"
)

type PointsSvc struct {
	log     *logrus.Entry
	gateway IGateway
}

func NewPointsSvc(gateway IGateway, log *logrus.Entry) *PointsSvc {
	return &PointsSvc{gateway: gateway, log: log}
}

// Retrieves datapoints for all of the inventory objects in the passed-in VsphereInfo for the number of 20-second
// intervals indicated by the passed-in numSamplesReqd. Also returns the most recent sample time for the returned points.
func (svc *PointsSvc) RetrievePoints(vsInfo *model.VsphereInfo, numSamplesReqd int32) ([]*datapoint.Datapoint, time.Time) {
	perf, err := svc.gateway.queryPerf(vsInfo.Inv.Objects, numSamplesReqd)
	if err != nil {
		svc.log.WithError(err).Error("queryPerf failed")
		return nil, time.Time{}
	}
	var latestSampleTime time.Time
	var dps []*datapoint.Datapoint
	for _, baseMetric := range perf.Returnval {
		perfEntityMetric, ok := baseMetric.(*types.PerfEntityMetric)
		if !ok {
			svc.log.WithField("baseMetric", baseMetric).Error("Type coersion to PerfEntityMetric failed")
			continue
		}
		if latestSampleTime.IsZero() {
			latestSampleTime = perfEntityMetric.SampleInfo[len(perfEntityMetric.SampleInfo)-1].Timestamp
		}
		for _, metric := range perfEntityMetric.Value {
			intSeries, ok := metric.(*types.PerfMetricIntSeries)
			if !ok {
				svc.log.WithField("metric", metric).Error("Type coersion to PerfMetricIntSeries failed")
				continue
			}

			metricName, metricType := lookupCachedMetricData(vsInfo, intSeries)

			dims, ok := vsInfo.Inv.DimensionMap[perfEntityMetric.Entity.Value]
			if !ok {
				dims = map[string]string{}
			}

			if len(intSeries.Value) > 0 && intSeries.Value[0] > 0 {
				svc.log.Debugf("metric = %s, values = %v", metricName, intSeries.Value)
			}

			for i, value := range intSeries.Value {
				dps = append(dps, datapoint.New(
					metricName,
					dims,
					datapoint.NewIntValue(value),
					metricType,
					perfEntityMetric.SampleInfo[i].Timestamp,
				))
			}
		}
	}
	return dps, latestSampleTime
}

// Lookup the cached metric name and metric type that was generated on VsphereInfo retrieval.
func lookupCachedMetricData(vsInfo *model.VsphereInfo, intSeries *types.PerfMetricIntSeries) (string, datapoint.MetricType) {
	metricInfo := vsInfo.PerfCounterIndex[intSeries.Id.CounterId]
	metricName := metricInfo.MetricName
	metricType := statsTypeToMetricType(metricInfo.PerfCounterInfo.StatsType)
	return metricName, metricType
}

func statsTypeToMetricType(statsType types.PerfStatsType) datapoint.MetricType {
	if statsType == types.PerfStatsTypeDelta {
		return datapoint.Counter
	}
	return datapoint.Gauge
}
