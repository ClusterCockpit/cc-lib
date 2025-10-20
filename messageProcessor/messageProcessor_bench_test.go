package messageprocessor

import (
	"encoding/json"
	"testing"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
)

func BenchmarkProcessingS1R1Match0(b *testing.B) {

	mp, err := NewMessageProcessor()
	if err != nil {
		b.Error(err.Error())
		return
	}
	err = mp.FromConfigJSON(json.RawMessage(`
		{	
			"move_meta_to_tag_if": [
				{"if" : "name == 'match'", "key": "unit", "value": "unit_renamed"}
			]
		}`))

	for range b.N {
		m, err := lp.NewMetric("nomatch", map[string]string{"type": "node", "type-id": "0", "hostname": "myhost"}, map[string]string{"unit": "myunit"}, 1.0, time.Now())
		if err == nil {
			b.StartTimer()
			mp.ProcessMessage(m)
			b.StopTimer()
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/message")
}

func BenchmarkProcessingS1R1Match1(b *testing.B) {

	mp, err := NewMessageProcessor()
	if err != nil {
		b.Error(err.Error())
		return
	}
	err = mp.FromConfigJSON(json.RawMessage(`
		{
			"move_meta_to_tag_if": [
				{"if" : "name == 'match'", "key": "unit", "value": "unit_renamed"}
			]
		}`))
	if err != nil {
		b.Error(err.Error())
		return
	}

	for range b.N {
		m, err := lp.NewMetric("match", map[string]string{"type": "node", "type-id": "0", "hostname": "myhost"}, map[string]string{"unit": "myunit"}, 1.0, time.Now())
		if err == nil {
			b.StartTimer()
			mp.ProcessMessage(m)
			b.StopTimer()
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/message")
}

func BenchmarkProcessingS1R2Match0(b *testing.B) {

	mp, err := NewMessageProcessor()
	if err != nil {
		b.Error(err.Error())
		return
	}
	err = mp.FromConfigJSON(json.RawMessage(`
		{	
			"move_meta_to_tag_if": [
				{"if" : "name == 'match'", "key": "unit", "value": "unit_renamed"},
				{"if" : "name == 'match'", "key": "foo", "value": "bar"}
			]
		}`))

	for range b.N {
		m, err := lp.NewMetric("nomatch", map[string]string{"type": "node", "type-id": "0", "hostname": "myhost"}, map[string]string{"unit": "myunit"}, 1.0, time.Now())
		if err == nil {
			b.StartTimer()
			mp.ProcessMessage(m)
			b.StopTimer()
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/message")
}

func BenchmarkProcessingS1R2Match1First(b *testing.B) {

	mp, err := NewMessageProcessor()
	if err != nil {
		b.Error(err.Error())
		return
	}
	err = mp.FromConfigJSON(json.RawMessage(`
		{	
			"move_meta_to_tag_if": [
				{"if" : "name == 'match'", "key": "unit", "value": "unit_renamed"},
				{"if" : "name == 'nomatch'", "key": "foo", "value": "bar"}
			]
		}`))

	for range b.N {
		m, err := lp.NewMetric("match", map[string]string{"type": "node", "type-id": "0", "hostname": "myhost"}, map[string]string{"unit": "myunit"}, 1.0, time.Now())
		if err == nil {
			b.StartTimer()
			mp.ProcessMessage(m)
			b.StopTimer()
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/message")
}

func BenchmarkProcessingS1R2Match1Second(b *testing.B) {

	mp, err := NewMessageProcessor()
	if err != nil {
		b.Error(err.Error())
		return
	}
	err = mp.FromConfigJSON(json.RawMessage(`
		{	
			"move_meta_to_tag_if": [
				{"if" : "name == 'nomatch'", "key": "foo", "value": "bar"},
				{"if" : "name == 'match'", "key": "unit", "value": "unit_renamed"}
			]
		}`))

	for range b.N {
		m, err := lp.NewMetric("match", map[string]string{"type": "node", "type-id": "0", "hostname": "myhost"}, map[string]string{"unit": "myunit"}, 1.0, time.Now())
		if err == nil {
			b.StartTimer()
			mp.ProcessMessage(m)
			b.StopTimer()
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/message")
}

func BenchmarkProcessingS2R2Match0(b *testing.B) {

	mp, err := NewMessageProcessor()
	if err != nil {
		b.Error(err.Error())
		return
	}
	err = mp.FromConfigJSON(json.RawMessage(`
		{	
			"move_meta_to_tag_if": [
				{"if" : "name == 'match'", "key": "unit", "value": "unit_renamed"}
			],
			"add_tags_if": [
				{"if" : "name == 'match'", "key": "mytagkey", "value": "mytagvalue"}
			]
		}`))

	for range b.N {
		m, err := lp.NewMetric("nomatch", map[string]string{"type": "node", "type-id": "0", "hostname": "myhost"}, map[string]string{"unit": "myunit"}, 1.0, time.Now())
		if err == nil {
			b.StartTimer()
			mp.ProcessMessage(m)
			b.StopTimer()
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/message")
}

func BenchmarkProcessingS2R2Match1First(b *testing.B) {

	mp, err := NewMessageProcessor()
	if err != nil {
		b.Error(err.Error())
		return
	}
	err = mp.FromConfigJSON(json.RawMessage(`
		{	
			"move_meta_to_tag_if": [
				{"if" : "name == 'match'", "key": "unit", "value": "unit_renamed"}
			],
			"add_tags_if": [
				{"if" : "name == 'nomatch'", "key": "mytagkey", "value": "mytagvalue"}
			],
			"stages": [
				"move_meta_to_tags",
				"add_tag"
			]
		}`))

	for range b.N {
		m, err := lp.NewMetric("match", map[string]string{"type": "node", "type-id": "0", "hostname": "myhost"}, map[string]string{"unit": "myunit"}, 1.0, time.Now())
		if err == nil {
			b.StartTimer()
			mp.ProcessMessage(m)
			b.StopTimer()
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/message")
}

func BenchmarkProcessingS2R2Match1Second(b *testing.B) {

	mp, err := NewMessageProcessor()
	if err != nil {
		b.Error(err.Error())
		return
	}
	err = mp.FromConfigJSON(json.RawMessage(`
		{	
			"move_meta_to_tag_if": [
				{"if" : "name == 'match'", "key": "unit", "value": "unit_renamed"}
			],
			"add_tags_if": [
				{"if" : "name == 'nomatch'", "key": "mytagkey", "value": "mytagvalue"}
			],
			"stages": [
				"add_tag",
				"move_meta_to_tags"
			]
		}`))

	for range b.N {
		m, err := lp.NewMetric("match", map[string]string{"type": "node", "type-id": "0", "hostname": "myhost"}, map[string]string{"unit": "myunit"}, 1.0, time.Now())
		if err == nil {
			b.StartTimer()
			mp.ProcessMessage(m)
			b.StopTimer()
		}
	}

	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N), "ns/message")
}
