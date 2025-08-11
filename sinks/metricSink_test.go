package sinks

import (
	"fmt"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
)

func gen_messages(numMessage int) ([]lp.CCMessage, error) {
	out := make([]lp.CCMessage, 0, numMessage)
	tags := map[string]string{
		"type": "node",
	}
	for i := 0; i < numMessage; i++ {
		x, err := lp.NewMetric(fmt.Sprintf("testmetric%d", i), tags, nil, 42+(0.66*float64(i)), time.Now())
		if err == nil {
			out = append(out, x)
		} else {
			return nil, err
		}
	}
	return out, nil
}
