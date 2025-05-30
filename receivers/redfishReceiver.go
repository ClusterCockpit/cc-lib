//go:build linux

// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)
package receivers

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	cclog "github.com/ClusterCockpit/cc-lib/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/ccMessage"
	"github.com/ClusterCockpit/cc-lib/hostlist"
	mp "github.com/ClusterCockpit/cc-lib/messageProcessor"

	// See: https://pkg.go.dev/github.com/stmcginnis/gofish
	"github.com/stmcginnis/gofish"
	"github.com/stmcginnis/gofish/common"
	"github.com/stmcginnis/gofish/redfish"
)

type RedfishReceiverClientConfig struct {
	// Hostname the redfish service belongs to
	Hostname string

	// is metric excluded globally or per client
	isExcluded map[string](bool)

	doPowerMetric      bool
	doProcessorMetrics bool
	doSensors          bool
	doThermalMetrics   bool

	skipProcessorMetricsURL map[string]bool

	// readSensorURLs stores for each chassis ID a list of sensor URLs to read
	readSensorURLs map[string][]string

	gofish gofish.ClientConfig

	mp mp.MessageProcessor
}

// RedfishReceiver configuration:
type RedfishReceiver struct {
	receiver

	config struct {
		defaultReceiverConfig
		fanout      int
		Interval    time.Duration
		HttpTimeout time.Duration

		// Client config for each redfish service
		ClientConfigs []RedfishReceiverClientConfig
	}

	done chan bool      // channel to finish / stop redfish receiver
	wg   sync.WaitGroup // wait group for redfish receiver
}

// deleteEmptyTags removes tags or meta data tags with empty value
func deleteEmptyTags(tags map[string]string) {
	maps.DeleteFunc(
		tags,
		func(key string, value string) bool {
			return value == ""
		},
	)
}

// setMetricValue sets the value entry in the fields map
func setMetricValue(value any) map[string]interface{} {
	return map[string]interface{}{
		"value": value,
	}
}

// sendMetric sends the metric through the sink channel
func (r *RedfishReceiver) sendMetric(mp mp.MessageProcessor, name string, tags map[string]string, meta map[string]string, value any, timestamp time.Time) {
	deleteEmptyTags(tags)
	deleteEmptyTags(meta)
	y, err := lp.NewMessage(name, tags, meta, setMetricValue(value), timestamp)
	if err == nil {
		mc, err := mp.ProcessMessage(y)
		if err == nil && mc != nil {
			m, err := r.mp.ProcessMessage(mc)
			if err == nil && m != nil {
				r.sink <- m
			}
		}
	}
}

// readSensors reads sensors from a redfish device
// See: https://redfish.dmtf.org/schemas/v1/Sensor.json
// Redfish URI: /redfish/v1/Chassis/{ChassisId}/Sensors/{SensorId}
func (r *RedfishReceiver) readSensors(
	clientConfig *RedfishReceiverClientConfig,
	chassis *redfish.Chassis,
) error {
	writeTemperatureSensor := func(sensor *redfish.Sensor) {
		tags := map[string]string{
			"hostname": clientConfig.Hostname,
			"type":     "node",
			// ChassisType shall indicate the physical form factor for the type of chassis
			"chassis_typ": string(chassis.ChassisType),
			// Chassis name
			"chassis_name": chassis.Name,
			// ID uniquely identifies the resource
			"sensor_id": sensor.ID,
			// The area or device to which this sensor measurement applies
			"temperature_physical_context": string(sensor.PhysicalContext),
			// Name
			"temperature_name": sensor.Name,
		}

		// Set meta data tags
		meta := map[string]string{
			"source": r.name,
			"group":  "Temperature",
			"unit":   "degC",
		}

		r.sendMetric(clientConfig.mp, "temperature", tags, meta, sensor.Reading, time.Now())
	}

	writeFanSpeedSensor := func(sensor *redfish.Sensor) {
		tags := map[string]string{
			"hostname": clientConfig.Hostname,
			"type":     "node",
			// ChassisType shall indicate the physical form factor for the type of chassis
			"chassis_typ": string(chassis.ChassisType),
			// Chassis name
			"chassis_name": chassis.Name,
			// ID uniquely identifies the resource
			"sensor_id": sensor.ID,
			// The area or device to which this sensor measurement applies
			"fan_physical_context": string(sensor.PhysicalContext),
			// Name
			"fan_name": sensor.Name,
		}

		// Set meta data tags
		meta := map[string]string{
			"source": r.name,
			"group":  "FanSpeed",
			"unit":   string(sensor.ReadingUnits),
		}

		r.sendMetric(clientConfig.mp, "fan_speed", tags, meta, sensor.Reading, time.Now())
	}

	writePowerSensor := func(sensor *redfish.Sensor) {
		// Set tags
		tags := map[string]string{
			"hostname": clientConfig.Hostname,
			"type":     "node",
			// ChassisType shall indicate the physical form factor for the type of chassis
			"chassis_typ": string(chassis.ChassisType),
			// Chassis name
			"chassis_name": chassis.Name,
			// ID uniquely identifies the resource
			"sensor_id": sensor.ID,
			// The area or device to which this sensor measurement applies
			"power_physical_context": string(sensor.PhysicalContext),
			// Name
			"power_name": sensor.Name,
		}

		// Set meta data tags
		meta := map[string]string{
			"source": r.name,
			"group":  "Energy",
			"unit":   "watts",
		}

		r.sendMetric(clientConfig.mp, "power", tags, meta, sensor.Reading, time.Now())
	}

	if _, ok := clientConfig.readSensorURLs[chassis.ID]; !ok {
		// First time run of read sensors for this chassis

		clientConfig.readSensorURLs[chassis.ID] = make([]string, 0)

		// Get sensor information for this chassis
		sensors, err := chassis.Sensors()
		if err != nil {
			return fmt.Errorf("readSensors: chassis.Sensors() failed: %v", err)
		}

		// Skip empty sensors information
		if sensors == nil {
			return nil
		}

		for _, sensor := range sensors {

			// Skip all sensors which are not in enabled state or which are unhealthy
			if sensor.Status.State != common.EnabledState || sensor.Status.Health != common.OKHealth {
				continue
			}

			// Skip sensors with missing readings units or type
			if sensor.ReadingUnits == "" || sensor.ReadingType == "" {
				continue
			}

			// Power readings
			if (sensor.ReadingType == redfish.PowerReadingType && sensor.ReadingUnits == "Watts") ||
				(sensor.ReadingType == redfish.CurrentReadingType && sensor.ReadingUnits == "Watts") {
				if clientConfig.isExcluded["power"] {
					continue
				}

				clientConfig.readSensorURLs[chassis.ID] = append(clientConfig.readSensorURLs[chassis.ID], sensor.ODataID)
				writePowerSensor(sensor)
				continue
			}

			// Fan speed readings
			if (sensor.ReadingType == redfish.AirFlowReadingType && sensor.ReadingUnits == "RPM") ||
				(sensor.ReadingType == redfish.AirFlowReadingType && sensor.ReadingUnits == "Percent") {
				// Skip, when fan_speed metric is excluded
				if clientConfig.isExcluded["fan_speed"] {
					continue
				}

				clientConfig.readSensorURLs[chassis.ID] = append(clientConfig.readSensorURLs[chassis.ID], sensor.ODataID)
				writeFanSpeedSensor(sensor)
			}

			// Temperature readings
			if sensor.ReadingType == redfish.TemperatureReadingType && sensor.ReadingUnits == "C" {
				if clientConfig.isExcluded["temperature"] {
					continue
				}

				clientConfig.readSensorURLs[chassis.ID] = append(clientConfig.readSensorURLs[chassis.ID], sensor.ODataID)
				writeTemperatureSensor(sensor)
				continue
			}
		}
	} else {
		common.CollectCollection(
			func(uri string) {
				sensor, err := redfish.GetSensor(chassis.GetClient(), uri)
				if err != nil {
					cclog.ComponentError(r.name, "redfish.GetSensor() for uri '", uri, "' failed")
				}

				// Power readings
				if (sensor.ReadingType == redfish.PowerReadingType && sensor.ReadingUnits == "Watts") ||
					(sensor.ReadingType == redfish.CurrentReadingType && sensor.ReadingUnits == "Watts") {

					writePowerSensor(sensor)
					return
				}

				// Fan speed readings
				if (sensor.ReadingType == redfish.AirFlowReadingType && sensor.ReadingUnits == "RPM") ||
					(sensor.ReadingType == redfish.AirFlowReadingType && sensor.ReadingUnits == "Percent") {

					writeFanSpeedSensor(sensor)
					return
				}

				// Temperature readings
				if sensor.ReadingType == redfish.TemperatureReadingType && sensor.ReadingUnits == "C" {

					writeTemperatureSensor(sensor)
					return
				}
			},
			clientConfig.readSensorURLs[chassis.ID])
	}
	return nil
}

// readThermalMetrics reads thermal metrics from a redfish device
// See: https://redfish.dmtf.org/schemas/v1/Thermal.json
// Redfish URI: /redfish/v1/Chassis/{ChassisId}/Thermal
// -> deprecated in favor of the ThermalSubsystem schema
// -> on Lenovo servers /redfish/v1/Chassis/{ChassisId}/ThermalSubsystem/ThermalMetrics links to /redfish/v1/Chassis/{ChassisId}/Sensors/{SensorId}
func (r *RedfishReceiver) readThermalMetrics(
	clientConfig *RedfishReceiverClientConfig,
	chassis *redfish.Chassis,
) error {
	// Get thermal information for each chassis
	thermal, err := chassis.Thermal()
	if err != nil {
		return fmt.Errorf("readMetrics: chassis.Thermal() failed: %v", err)
	}

	// Skip empty thermal information
	if thermal == nil {
		return nil
	}

	timestamp := time.Now()

	for _, temperature := range thermal.Temperatures {

		// Skip, when temperature metric is excluded
		if clientConfig.isExcluded["temperature"] {
			break
		}

		// Skip all temperatures which are not in enabled state
		if temperature.Status.State != "" && temperature.Status.State != common.EnabledState {
			continue
		}

		tags := map[string]string{
			"hostname": clientConfig.Hostname,
			"type":     "node",
			// ChassisType shall indicate the physical form factor for the type of chassis
			"chassis_typ": string(chassis.ChassisType),
			// Chassis name
			"chassis_name": chassis.Name,
			// ID uniquely identifies the resource
			"temperature_id": temperature.ID,
			// MemberID shall uniquely identify the member within the collection. For
			// services supporting Redfish v1.6 or higher, this value shall be the
			// zero-based array index.
			"temperature_member_id": temperature.MemberID,
			// PhysicalContext shall be a description of the affected device or region
			// within the chassis to which this temperature measurement applies
			"temperature_physical_context": string(temperature.PhysicalContext),
			// Name
			"temperature_name": temperature.Name,
		}

		// Set meta data tags
		meta := map[string]string{
			"source": r.name,
			"group":  "Temperature",
			"unit":   "degC",
		}

		// ReadingCelsius shall be the current value of the temperature sensor's reading.
		value := temperature.ReadingCelsius

		r.sendMetric(clientConfig.mp, "temperature", tags, meta, value, timestamp)
	}

	for _, fan := range thermal.Fans {
		// Skip, when fan_speed metric is excluded
		if clientConfig.isExcluded["fan_speed"] {
			break
		}

		// Skip all fans which are not in enabled state
		if fan.Status.State != common.EnabledState {
			continue
		}

		tags := map[string]string{
			"hostname": clientConfig.Hostname,
			"type":     "node",
			// ChassisType shall indicate the physical form factor for the type of chassis
			"chassis_typ": string(chassis.ChassisType),
			// Chassis name
			"chassis_name": chassis.Name,
			// ID uniquely identifies the resource
			"fan_id": fan.ID,
			// MemberID shall uniquely identify the member within the collection. For
			// services supporting Redfish v1.6 or higher, this value shall be the
			// zero-based array index.
			"fan_member_id": fan.MemberID,
			// PhysicalContext shall be a description of the affected device or region
			// within the chassis to which this fan is associated
			"fan_physical_context": string(fan.PhysicalContext),
			// Name
			"fan_name": fan.Name,
		}

		// Set meta data tags
		meta := map[string]string{
			"source": r.name,
			"group":  "FanSpeed",
			"unit":   string(fan.ReadingUnits),
		}

		r.sendMetric(clientConfig.mp, "fan_speed", tags, meta, fan.Reading, timestamp)
	}

	return nil
}

// readPowerMetrics reads power metrics from a redfish device
// See: https://redfish.dmtf.org/schemas/v1/Power.json
// Redfish URI: /redfish/v1/Chassis/{ChassisId}/Power
// -> deprecated in favor of the PowerSubsystem schema
func (r *RedfishReceiver) readPowerMetrics(
	clientConfig *RedfishReceiverClientConfig,
	chassis *redfish.Chassis,
) error {
	// Get power information for each chassis
	power, err := chassis.Power()
	if err != nil {
		return fmt.Errorf("readMetrics: chassis.Power() failed: %v", err)
	}

	// Skip empty power information
	if power == nil {
		return nil
	}

	timestamp := time.Now()

	// Read min, max and average consumed watts for each power control
	for _, pc := range power.PowerControl {

		// Skip all power controls which are not in enabled state
		if pc.Status.State != "" && pc.Status.State != common.EnabledState {
			continue
		}

		// Map of collected metrics
		metrics := make(map[string]float32)

		// PowerConsumedWatts shall represent the actual power being consumed (in
		// Watts) by the chassis
		if !clientConfig.isExcluded["consumed_watts"] {
			metrics["consumed_watts"] = pc.PowerConsumedWatts
		}
		// AverageConsumedWatts shall represent the
		// average power level that occurred averaged over the last IntervalInMin
		// minutes.
		if !clientConfig.isExcluded["average_consumed_watts"] {
			metrics["average_consumed_watts"] = pc.PowerMetrics.AverageConsumedWatts
		}
		// MinConsumedWatts shall represent the
		// minimum power level in watts that occurred within the last
		// IntervalInMin minutes.
		if !clientConfig.isExcluded["min_consumed_watts"] {
			metrics["min_consumed_watts"] = pc.PowerMetrics.MinConsumedWatts
		}
		// MaxConsumedWatts shall represent the
		// maximum power level in watts that occurred within the last
		// IntervalInMin minutes
		if !clientConfig.isExcluded["max_consumed_watts"] {
			metrics["max_consumed_watts"] = pc.PowerMetrics.MaxConsumedWatts
		}
		// IntervalInMin shall represent the time interval (or window), in minutes,
		// in which the PowerMetrics properties are measured over.
		// Should be an integer, but some Dell implementations return as a float
		intervalInMin := strconv.FormatFloat(
			float64(pc.PowerMetrics.IntervalInMin), 'f', -1, 32)

		// Set tags
		tags := map[string]string{
			"hostname": clientConfig.Hostname,
			"type":     "node",
			// ChassisType shall indicate the physical form factor for the type of chassis
			"chassis_typ": string(chassis.ChassisType),
			// Chassis name
			"chassis_name": chassis.Name,
			// ID uniquely identifies the resource
			"power_control_id": pc.ID,
			// MemberID shall uniquely identify the member within the collection. For
			// services supporting Redfish v1.6 or higher, this value shall be the
			// zero-based array index.
			"power_control_member_id": pc.MemberID,
			// PhysicalContext shall be a description of the affected device(s) or region
			// within the chassis to which this power control applies.
			"power_control_physical_context": string(pc.PhysicalContext),
			// Name
			"power_control_name": pc.Name,
		}

		// Set meta data tags
		meta := map[string]string{
			"source":              r.name,
			"group":               "Energy",
			"interval_in_minutes": intervalInMin,
			"unit":                "watts",
		}

		for name, value := range metrics {
			r.sendMetric(clientConfig.mp, name, tags, meta, value, timestamp)
		}
	}

	return nil
}

// readProcessorMetrics reads processor metrics from a redfish device
// See: https://redfish.dmtf.org/schemas/v1/ProcessorMetrics.json
// Redfish URI: /redfish/v1/Systems/{ComputerSystemId}/Processors/{ProcessorId}/ProcessorMetrics
func (r *RedfishReceiver) readProcessorMetrics(
	clientConfig *RedfishReceiverClientConfig,
	processor *redfish.Processor,
) error {
	timestamp := time.Now()

	// URL to processor metrics
	URL := processor.ODataID + "/ProcessorMetrics"

	// Skip previously detected non existing URLs
	if clientConfig.skipProcessorMetricsURL[URL] {
		return nil
	}

	resp, err := processor.GetClient().Get(URL)
	if err != nil {
		// Skip non existing URLs
		if statusCode := err.(*common.Error).HTTPReturnedStatusCode; statusCode == http.StatusNotFound {
			clientConfig.skipProcessorMetricsURL[URL] = true
			return nil
		}

		return fmt.Errorf("processor.GetClient().Get(%v) failed: %+w", URL, err)
	}

	var processorMetrics struct {
		common.Entity
		ODataType   string `json:"@odata.type"`
		ODataEtag   string `json:"@odata.etag"`
		Description string `json:"Description"`
		// This property shall contain the power, in watts, that the processor has consumed.
		ConsumedPowerWatt float32 `json:"ConsumedPowerWatt"`
		// This property shall contain the temperature, in Celsius, of the processor.
		TemperatureCelsius float32 `json:"TemperatureCelsius"`
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("unable to read response body for processor metrics: %+w", err)
	}
	err = json.Unmarshal(body, &processorMetrics)
	if err != nil {
		return fmt.Errorf(
			"unable to unmarshal JSON='%s' for processor metrics: %+w",
			string(body),
			err,
		)
	}

	// Set tags
	tags := map[string]string{
		"hostname": clientConfig.Hostname,
		"type":     "socket",
		// ProcessorType shall contain the string which identifies the type of processor contained in this Socket
		"processor_typ": string(processor.ProcessorType),
		// Processor name
		"processor_name": processor.Name,
		// ID uniquely identifies the resource
		"processor_id": processor.ID,
	}

	// Set meta data tags
	metaPower := map[string]string{
		"source": r.name,
		"group":  "Energy",
		"unit":   "watts",
	}

	namePower := "consumed_power"

	if !clientConfig.isExcluded[namePower] &&
		// Some servers return "ConsumedPowerWatt":65535 instead of "ConsumedPowerWatt":null
		processorMetrics.ConsumedPowerWatt != 65535 {
		r.sendMetric(clientConfig.mp, namePower, tags, metaPower, processorMetrics.ConsumedPowerWatt, timestamp)
	}
	// Set meta data tags
	metaThermal := map[string]string{
		"source": r.name,
		"group":  "Temperature",
		"unit":   "degC",
	}

	nameThermal := "temperature"

	if !clientConfig.isExcluded[nameThermal] {
		r.sendMetric(clientConfig.mp, nameThermal, tags, metaThermal, processorMetrics.TemperatureCelsius, timestamp)
	}
	return nil
}

// readMetrics reads redfish thermal, power and processor metrics from the redfish device
// configured in clientConfig
func (r *RedfishReceiver) readMetrics(clientConfig *RedfishReceiverClientConfig) error {
	// Connect to redfish service
	c, err := gofish.Connect(clientConfig.gofish)
	if err != nil {
		return fmt.Errorf(
			"readMetrics: gofish.Connect({Username: %v, Endpoint: %v, BasicAuth: %v, HttpTimeout: %v, HttpInsecure: %v}) failed: %v",
			clientConfig.gofish.Username,
			clientConfig.gofish.Endpoint,
			clientConfig.gofish.BasicAuth,
			clientConfig.gofish.HTTPClient.Timeout,
			clientConfig.gofish.HTTPClient.Transport.(*http.Transport).TLSClientConfig.InsecureSkipVerify,
			err)
	}
	defer c.Logout()

	// Create a session, when required
	if _, err = c.GetSession(); err != nil {
		c, err = c.CloneWithSession()
		if err != nil {
			return fmt.Errorf("readMetrics: Failed to create a session: %+w", err)
		}
	}

	// Get all chassis managed by this service
	isChassisListRequired := clientConfig.doSensors ||
		clientConfig.doThermalMetrics ||
		clientConfig.doPowerMetric
	var chassisList []*redfish.Chassis
	if isChassisListRequired {
		chassisList, err = c.Service.Chassis()
		if err != nil {
			return fmt.Errorf("readMetrics: c.Service.Chassis() failed: %v", err)
		}
	}

	// Get all computer systems managed by this service
	isComputerSystemListRequired := clientConfig.doProcessorMetrics
	var computerSystemList []*redfish.ComputerSystem
	if isComputerSystemListRequired {
		computerSystemList, err = c.Service.Systems()
		if err != nil {
			return fmt.Errorf("readMetrics: c.Service.Systems() failed: %v", err)
		}
	}

	// Read sensors
	if clientConfig.doSensors {
		for _, chassis := range chassisList {
			err := r.readSensors(clientConfig, chassis)
			if err != nil {
				return err
			}
		}
	}

	// read thermal metrics
	if clientConfig.doThermalMetrics {
		for _, chassis := range chassisList {
			err := r.readThermalMetrics(clientConfig, chassis)
			if err != nil {
				return err
			}
		}
	}

	// read power metrics
	if clientConfig.doPowerMetric {
		for _, chassis := range chassisList {
			err = r.readPowerMetrics(clientConfig, chassis)
			if err != nil {
				return err
			}
		}
	}

	// read processor metrics
	if clientConfig.doProcessorMetrics {
		// loop for all computer systems
		for _, system := range computerSystemList {

			// loop for all processors
			processors, err := system.Processors()
			if err != nil {
				return fmt.Errorf("readMetrics: system.Processors() failed: %v", err)
			}
			for _, processor := range processors {
				err := r.readProcessorMetrics(clientConfig, processor)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// doReadMetrics reads metrics from all configure redfish devices.
// To compensate latencies of the Redfish devices a fanout is used.
func (r *RedfishReceiver) doReadMetric() {
	// Create wait group and input channel for workers
	var workerWaitGroup sync.WaitGroup
	workerInput := make(chan *RedfishReceiverClientConfig, r.config.fanout)

	// Create worker go routines
	for i := 0; i < r.config.fanout; i++ {
		// Increment worker wait group counter
		workerWaitGroup.Add(1)
		go func() {
			// Decrement worker wait group counter
			defer workerWaitGroup.Done()

			// Read power metrics for each client config
			for clientConfig := range workerInput {
				err := r.readMetrics(clientConfig)
				if err != nil {
					cclog.ComponentError(r.name, err)
				}
			}
		}()
	}

	// Distribute client configs to workers
	for i := range r.config.ClientConfigs {
		// Check done channel status
		select {
		case workerInput <- &r.config.ClientConfigs[i]:
		case <-r.done:
			// process done event
			// Stop workers, clear channel and wait for all workers to finish
			close(workerInput)
			for range workerInput {
			}
			workerWaitGroup.Wait()
			return
		}
	}

	// Stop workers and wait for all workers to finish
	close(workerInput)
	workerWaitGroup.Wait()
}

// Start starts the redfish receiver
func (r *RedfishReceiver) Start() {
	cclog.ComponentDebug(r.name, "START")

	// Start redfish receiver
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		// Create ticker
		ticker := time.NewTicker(r.config.Interval)
		defer ticker.Stop()

		for {
			r.doReadMetric()

			select {
			case tickerTime := <-ticker.C:
				// Check if we missed the ticker event
				if since := time.Since(tickerTime); since > 5*time.Second {
					cclog.ComponentInfo(r.name, "Missed ticker event for more then", since)
				}

				// process ticker event -> continue
				continue
			case <-r.done:
				// process done event
				return
			}
		}
	}()

	cclog.ComponentDebug(r.name, "STARTED")
}

// Close closes the redfish receiver
func (r *RedfishReceiver) Close() {
	cclog.ComponentDebug(r.name, "CLOSE")

	// Send the signal and wait
	close(r.done)
	r.wg.Wait()

	cclog.ComponentDebug(r.name, "DONE")
}

// NewRedfishReceiver creates a new instance of the redfish receiver
// Initialize the receiver by giving it a name and reading in the config JSON
func NewRedfishReceiver(name string, config json.RawMessage) (Receiver, error) {
	var err error
	r := new(RedfishReceiver)

	// Config options from config file
	configJSON := struct {
		Type             string          `json:"type"`
		MessageProcessor json.RawMessage `json:"process_messages,omitempty"`

		// Maximum number of simultaneous redfish connections (default: 64)
		Fanout int `json:"fanout,omitempty"`
		// How often the redfish power metrics should be read and send to the sink (default: 30 s)
		IntervalString string `json:"interval,omitempty"`

		// Control whether a client verifies the server's certificate
		// (default: true == do not verify server's certificate)
		HttpInsecure bool `json:"http_insecure,omitempty"`
		// Time limit for requests made by this HTTP client (default: 10 s)
		HttpTimeoutString string `json:"http_timeout,omitempty"`

		// Default client username, password and endpoint
		Username *string `json:"username"` // User name to authenticate with
		Password *string `json:"password"` // Password to use for authentication
		Endpoint *string `json:"endpoint"` // URL of the redfish service

		// Globally disable collection of power, processor or thermal metrics
		DisablePowerMetrics     bool `json:"disable_power_metrics"`
		DisableProcessorMetrics bool `json:"disable_processor_metrics"`
		DisableSensors          bool `json:"disable_sensors"`
		DisableThermalMetrics   bool `json:"disable_thermal_metrics"`

		// Globally excluded metrics
		ExcludeMetrics []string `json:"exclude_metrics,omitempty"`

		ClientConfigs []struct {
			HostList string  `json:"host_list"` // List of hosts with the same client configuration
			Username *string `json:"username"`  // User name to authenticate with
			Password *string `json:"password"`  // Password to use for authentication
			Endpoint *string `json:"endpoint"`  // URL of the redfish service

			// Per client disable collection of power,processor or thermal metrics
			DisablePowerMetrics     bool `json:"disable_power_metrics"`
			DisableProcessorMetrics bool `json:"disable_processor_metrics"`
			DisableSensors          bool `json:"disable_sensors"`
			DisableThermalMetrics   bool `json:"disable_thermal_metrics"`

			// Per client excluded metrics
			ExcludeMetrics   []string        `json:"exclude_metrics,omitempty"`
			MessageProcessor json.RawMessage `json:"process_messages,omitempty"`
		} `json:"client_config"`
	}{
		// Set defaults values
		// Allow overwriting these defaults by reading config JSON
		Fanout:            64,
		IntervalString:    "30s",
		HttpTimeoutString: "10s",
		HttpInsecure:      true,
	}

	// Set name
	r.name = fmt.Sprintf("RedfishReceiver(%s)", name)

	// Create done channel
	r.done = make(chan bool)

	// Read the redfish receiver specific JSON config
	if len(config) > 0 {
		d := json.NewDecoder(bytes.NewReader(config))
		d.DisallowUnknownFields()
		if err := d.Decode(&configJSON); err != nil {
			cclog.ComponentError(r.name, "Error reading config:", err.Error())
			return nil, err
		}
	}
	p, err := mp.NewMessageProcessor()
	if err != nil {
		return nil, fmt.Errorf("initialization of message processor failed: %v", err.Error())
	}
	r.mp = p
	if len(r.config.MessageProcessor) > 0 {
		err = r.mp.FromConfigJSON(r.config.MessageProcessor)
		if err != nil {
			return nil, fmt.Errorf("failed parsing JSON for message processor: %v", err.Error())
		}
	}

	// Convert interval string representation to duration

	r.config.Interval, err = time.ParseDuration(configJSON.IntervalString)
	if err != nil {
		err := fmt.Errorf(
			"failed to parse duration string interval='%s': %w",
			configJSON.IntervalString,
			err,
		)
		cclog.Error(r.name, err)
		return nil, err
	}

	// HTTP timeout duration
	r.config.HttpTimeout, err = time.ParseDuration(configJSON.HttpTimeoutString)
	if err != nil {
		err := fmt.Errorf(
			"failed to parse duration string http_timeout='%s': %w",
			configJSON.HttpTimeoutString,
			err,
		)
		cclog.Error(r.name, err)
		return nil, err
	}

	// Create new http client
	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: configJSON.HttpInsecure,
	}
	httpClient := &http.Client{
		Timeout:   r.config.HttpTimeout,
		Transport: customTransport,
	}

	// Initialize client configurations
	r.config.ClientConfigs = make([]RedfishReceiverClientConfig, 0)

	// Create client config from JSON config
	for i := range configJSON.ClientConfigs {

		clientConfigJSON := &configJSON.ClientConfigs[i]

		// Redfish endpoint
		var endpoint_pattern string
		if clientConfigJSON.Endpoint != nil {
			endpoint_pattern = *clientConfigJSON.Endpoint
		} else if configJSON.Endpoint != nil {
			endpoint_pattern = *configJSON.Endpoint
		} else {
			err := fmt.Errorf("client config number %v requires endpoint", i)
			cclog.ComponentError(r.name, err)
			return nil, err
		}

		// Redfish username
		var username string
		if clientConfigJSON.Username != nil {
			username = *clientConfigJSON.Username
		} else if configJSON.Username != nil {
			username = *configJSON.Username
		} else {
			err := fmt.Errorf("client config number %v requires username", i)
			cclog.ComponentError(r.name, err)
			return nil, err
		}

		// Redfish password
		var password string
		if clientConfigJSON.Password != nil {
			password = *clientConfigJSON.Password
		} else if configJSON.Password != nil {
			password = *configJSON.Password
		} else {
			err := fmt.Errorf("client config number %v requires password", i)
			cclog.ComponentError(r.name, err)
			return nil, err
		}

		// Which metrics should be collected
		doPowerMetric := !(configJSON.DisablePowerMetrics ||
			clientConfigJSON.DisablePowerMetrics)
		doProcessorMetrics := !(configJSON.DisableProcessorMetrics ||
			clientConfigJSON.DisableProcessorMetrics)
		doSensors := !(configJSON.DisableSensors ||
			clientConfigJSON.DisableSensors)
		doThermalMetrics := !(configJSON.DisableThermalMetrics ||
			clientConfigJSON.DisableThermalMetrics)

		// Is metrics excluded globally or per client
		isExcluded := make(map[string]bool)
		for _, key := range clientConfigJSON.ExcludeMetrics {
			isExcluded[key] = true
		}
		for _, key := range configJSON.ExcludeMetrics {
			isExcluded[key] = true
		}
		p, err = mp.NewMessageProcessor()
		if err != nil {
			cclog.ComponentError(r.name, err.Error())
			return nil, err
		}
		if len(clientConfigJSON.MessageProcessor) > 0 {
			err = p.FromConfigJSON(clientConfigJSON.MessageProcessor)
			if err != nil {
				cclog.ComponentError(r.name, err.Error())
				return nil, err
			}
		}

		hostList, err := hostlist.Expand(clientConfigJSON.HostList)
		if err != nil {
			err := fmt.Errorf("client config number %d failed to parse host list %s: %v",
				i, clientConfigJSON.HostList, err)
			cclog.ComponentError(r.name, err)
			return nil, err
		}
		for _, host := range hostList {

			// Endpoint of the redfish service
			endpoint := strings.Replace(endpoint_pattern, "%h", host, -1)

			r.config.ClientConfigs = append(
				r.config.ClientConfigs,
				RedfishReceiverClientConfig{
					Hostname:                host,
					isExcluded:              isExcluded,
					doPowerMetric:           doPowerMetric,
					doProcessorMetrics:      doProcessorMetrics,
					doSensors:               doSensors,
					doThermalMetrics:        doThermalMetrics,
					skipProcessorMetricsURL: make(map[string]bool),
					readSensorURLs:          map[string][]string{},
					gofish: gofish.ClientConfig{
						Username:   username,
						Password:   password,
						Endpoint:   endpoint,
						HTTPClient: httpClient,
					},
					mp: p,
				})
		}

	}

	// Compute parallel fanout to use
	numClients := len(r.config.ClientConfigs)
	r.config.fanout = configJSON.Fanout
	if numClients < r.config.fanout {
		r.config.fanout = numClients
	}

	// Check that at least on client config exists
	if numClients == 0 {
		err := fmt.Errorf("at least one client config is required")
		cclog.ComponentError(r.name, err)
		return nil, err
	}

	// Check for duplicate client configurations
	isDuplicate := make(map[string]bool)
	for i := range r.config.ClientConfigs {
		host := r.config.ClientConfigs[i].Hostname
		if isDuplicate[host] {
			err := fmt.Errorf("found duplicate client config for host %s", host)
			cclog.ComponentError(r.name, err)
			return nil, err
		}
		isDuplicate[host] = true
	}

	// Give some basic info about redfish receiver status
	cclog.ComponentInfo(r.name, "Monitoring", numClients, "clients")
	cclog.ComponentInfo(r.name, "Monitoring interval:", r.config.Interval)
	cclog.ComponentInfo(r.name, "Monitoring parallel fanout:", r.config.fanout)

	return r, nil
}
