package vsphere

import (
	"context"
	"time"

	"github.com/signalfx/signalfx-agent/pkg/monitors"
	"github.com/signalfx/signalfx-agent/pkg/monitors/types"
	"github.com/signalfx/signalfx-agent/pkg/monitors/vsphere/model"
	"github.com/signalfx/signalfx-agent/pkg/utils"
	"github.com/sirupsen/logrus"
)

type Monitor struct {
	Output types.Output
	cancel func()
}

func init() {
	monitors.Register(
		&monitorMetadata,
		func() interface{} { return &Monitor{} },
		&model.Config{},
	)
}

func (m *Monitor) Configure(conf *model.Config) error {
	var ctx context.Context
	ctx, m.cancel = context.WithCancel(context.Background())
	log := logrus.WithField("monitorType", "vsphere")
	r := newRunner(ctx, log, conf, m)
	utils.RunOnInterval(ctx, r.run, time.Duration(conf.IntervalSeconds)*time.Second)
	return nil
}

func (m *Monitor) Shutdown() {
	if m.cancel != nil {
		m.cancel()
	}
}
