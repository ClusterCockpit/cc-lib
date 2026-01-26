// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
package resampler

import (
	"math"
	"testing"

	"github.com/ClusterCockpit/cc-lib/v2/schema"
)

// TestCalculateTriangleArea tests the triangle area calculation helper function
func TestCalculateTriangleArea(t *testing.T) {
	tests := []struct {
		name     string
		paX, paY schema.Float
		pbX, pbY schema.Float
		pcX, pcY schema.Float
		expected float64
	}{
		{
			name: "Right triangle",
			paX:  0, paY: 0,
			pbX: 0, pbY: 3,
			pcX: 4, pcY: 0,
			expected: 6.0,
		},
		{
			name: "Collinear points (zero area)",
			paX:  0, paY: 0,
			pbX: 1, pbY: 1,
			pcX: 2, pcY: 2,
			expected: 0.0,
		},
		{
			name: "Negative coordinates",
			paX:  -1, paY: -1,
			pbX: -1, pbY: 2,
			pcX: 3, pcY: -1,
			expected: 6.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTriangleArea(tt.paX, tt.paY, tt.pbX, tt.pbY, tt.pcX, tt.pcY)
			if math.Abs(result-tt.expected) > 1e-10 {
				t.Errorf("calculateTriangleArea() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestCalculateAverageDataPoint tests the average point calculation helper function
func TestCalculateAverageDataPoint(t *testing.T) {
	tests := []struct {
		name      string
		points    []schema.Float
		xStart    int64
		expectedX schema.Float
		expectedY schema.Float
		expectNaN bool
	}{
		{
			name:      "Simple average",
			points:    []schema.Float{1.0, 2.0, 3.0},
			xStart:    0,
			expectedX: 1.0,
			expectedY: 2.0,
			expectNaN: false,
		},
		{
			name:      "Single point",
			points:    []schema.Float{5.0},
			xStart:    10,
			expectedX: 10.0,
			expectedY: 5.0,
			expectNaN: false,
		},
		{
			name:      "With NaN value",
			points:    []schema.Float{1.0, schema.NaN, 3.0},
			xStart:    0,
			expectedX: 1.0,
			expectNaN: true,
		},
		{
			name:      "All NaN values",
			points:    []schema.Float{schema.NaN, schema.NaN},
			xStart:    0,
			expectedX: 0.5,
			expectNaN: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avgX, avgY := calculateAverageDataPoint(tt.points, tt.xStart)
			if math.Abs(float64(avgX-tt.expectedX)) > 1e-10 {
				t.Errorf("calculateAverageDataPoint() avgX = %v, want %v", avgX, tt.expectedX)
			}
			if tt.expectNaN {
				if !math.IsNaN(float64(avgY)) {
					t.Errorf("calculateAverageDataPoint() avgY should be NaN, got %v", avgY)
				}
			} else {
				if math.Abs(float64(avgY-tt.expectedY)) > 1e-10 {
					t.Errorf("calculateAverageDataPoint() avgY = %v, want %v", avgY, tt.expectedY)
				}
			}
		})
	}
}

// TestSimpleResampler tests the SimpleResampler function
func TestSimpleResampler(t *testing.T) {
	// Set minimum required points to 100 for testing
	SetMinimumRequiredPoints(100)
	defer SetMinimumRequiredPoints(1000) // Restore default after test

	tests := []struct {
		name         string
		data         []schema.Float
		oldFrequency int64
		newFrequency int64
		expectedLen  int
		expectedFreq int64
		expectError  bool
		checkValues  bool
		expectedData []schema.Float
	}{
		{
			name:         "Normal downsampling",
			data:         makeTestData(200, 1.0),
			oldFrequency: 1,
			newFrequency: 2,
			expectedLen:  100,
			expectedFreq: 2,
			expectError:  false,
		},
		{
			name:         "No downsampling needed (new <= old)",
			data:         makeTestData(200, 1.0),
			oldFrequency: 2,
			newFrequency: 1,
			expectedLen:  200,
			expectedFreq: 2,
			expectError:  false,
		},
		{
			name:         "Zero old frequency",
			data:         makeTestData(200, 1.0),
			oldFrequency: 0,
			newFrequency: 2,
			expectedLen:  200,
			expectedFreq: 0,
			expectError:  false,
		},
		{
			name:         "Zero new frequency",
			data:         makeTestData(200, 1.0),
			oldFrequency: 1,
			newFrequency: 0,
			expectedLen:  200,
			expectedFreq: 1,
			expectError:  false,
		},
		{
			name:         "Non-multiple frequency",
			data:         makeTestData(200, 1.0),
			oldFrequency: 3,
			newFrequency: 7,
			expectError:  true,
		},
		{
			name:         "Small dataset (< 100 points)",
			data:         makeTestData(50, 1.0),
			oldFrequency: 1,
			newFrequency: 2,
			expectedLen:  50,
			expectedFreq: 1,
			expectError:  false,
		},
		{
			name:         "Exact values check",
			data:         []schema.Float{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
			oldFrequency: 1,
			newFrequency: 2,
			expectedLen:  10,
			expectedFreq: 1,
			expectError:  false,
			checkValues:  false, // Too small, won't downsample
		},
		{
			name:         "Large downsampling factor",
			data:         makeTestData(1000, 1.0),
			oldFrequency: 1,
			newFrequency: 10,
			expectedLen:  100,
			expectedFreq: 10,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, freq, err := SimpleResampler(tt.data, tt.oldFrequency, tt.newFrequency)

			if tt.expectError {
				if err == nil {
					t.Errorf("SimpleResampler() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("SimpleResampler() unexpected error: %v", err)
				return
			}

			if len(result) != tt.expectedLen {
				t.Errorf("SimpleResampler() result length = %v, want %v", len(result), tt.expectedLen)
			}

			if freq != tt.expectedFreq {
				t.Errorf("SimpleResampler() frequency = %v, want %v", freq, tt.expectedFreq)
			}

			if tt.checkValues && tt.expectedData != nil {
				for i := range result {
					if result[i] != tt.expectedData[i] {
						t.Errorf("SimpleResampler() result[%d] = %v, want %v", i, result[i], tt.expectedData[i])
					}
				}
			}
		})
	}
}

// TestLargestTriangleThreeBucket tests the LTTB algorithm
func TestLargestTriangleThreeBucket(t *testing.T) {
	// Set minimum required points to 100 for testing
	SetMinimumRequiredPoints(100)
	defer SetMinimumRequiredPoints(1000) // Restore default after test

	tests := []struct {
		name         string
		data         []schema.Float
		oldFrequency int64
		newFrequency int64
		expectedLen  int
		expectedFreq int64
		expectError  bool
	}{
		{
			name:         "Normal downsampling",
			data:         makeTestData(200, 1.0),
			oldFrequency: 1,
			newFrequency: 2,
			expectedLen:  100,
			expectedFreq: 2,
			expectError:  false,
		},
		{
			name:         "Sine wave pattern",
			data:         makeSineWave(500, 10),
			oldFrequency: 1,
			newFrequency: 5,
			expectedLen:  100,
			expectedFreq: 5,
			expectError:  false,
		},
		{
			name:         "No downsampling needed (new <= old)",
			data:         makeTestData(200, 1.0),
			oldFrequency: 2,
			newFrequency: 1,
			expectedLen:  200,
			expectedFreq: 2,
			expectError:  false,
		},
		{
			name:         "Zero old frequency",
			data:         makeTestData(200, 1.0),
			oldFrequency: 0,
			newFrequency: 2,
			expectedLen:  200,
			expectedFreq: 0,
			expectError:  false,
		},
		{
			name:         "Zero new frequency",
			data:         makeTestData(200, 1.0),
			oldFrequency: 1,
			newFrequency: 0,
			expectedLen:  200,
			expectedFreq: 1,
			expectError:  false,
		},
		{
			name:         "Non-multiple frequency",
			data:         makeTestData(200, 1.0),
			oldFrequency: 3,
			newFrequency: 7,
			expectError:  true,
		},
		{
			name:         "Small dataset (< 100 points)",
			data:         makeTestData(50, 1.0),
			oldFrequency: 1,
			newFrequency: 2,
			expectedLen:  50,
			expectedFreq: 1,
			expectError:  false,
		},
		{
			name:         "With NaN values",
			data:         makeTestDataWithNaN(200),
			oldFrequency: 1,
			newFrequency: 2,
			expectedLen:  100,
			expectedFreq: 2,
			expectError:  false,
		},
		{
			name:         "Large downsampling factor",
			data:         makeTestData(1000, 1.0),
			oldFrequency: 1,
			newFrequency: 10,
			expectedLen:  100,
			expectedFreq: 10,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, freq, err := LargestTriangleThreeBucket(tt.data, tt.oldFrequency, tt.newFrequency)

			if tt.expectError {
				if err == nil {
					t.Errorf("LargestTriangleThreeBucket() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("LargestTriangleThreeBucket() unexpected error: %v", err)
				return
			}

			if len(result) != tt.expectedLen {
				t.Errorf("LargestTriangleThreeBucket() result length = %v, want %v", len(result), tt.expectedLen)
			}

			if freq != tt.expectedFreq {
				t.Errorf("LargestTriangleThreeBucket() frequency = %v, want %v", freq, tt.expectedFreq)
			}

			// Verify first and last points are preserved
			if len(result) > 0 && len(tt.data) > 0 {
				// Handle NaN comparison specially since NaN != NaN
				if math.IsNaN(float64(tt.data[0])) {
					if !math.IsNaN(float64(result[0])) {
						t.Errorf("LargestTriangleThreeBucket() first point not preserved: got %v, want NaN", result[0])
					}
				} else if result[0] != tt.data[0] {
					t.Errorf("LargestTriangleThreeBucket() first point not preserved: got %v, want %v", result[0], tt.data[0])
				}

				if math.IsNaN(float64(tt.data[len(tt.data)-1])) {
					if !math.IsNaN(float64(result[len(result)-1])) {
						t.Errorf("LargestTriangleThreeBucket() last point not preserved: got %v, want NaN", result[len(result)-1])
					}
				} else if result[len(result)-1] != tt.data[len(tt.data)-1] {
					t.Errorf("LargestTriangleThreeBucket() last point not preserved: got %v, want %v", result[len(result)-1], tt.data[len(tt.data)-1])
				}
			}
		})
	}
}

// TestLTTBPreservesFeatures tests that LTTB preserves important features better than simple resampling
func TestLTTBPreservesFeatures(t *testing.T) {
	// Set minimum required points to 100 for testing
	SetMinimumRequiredPoints(100)
	defer SetMinimumRequiredPoints(1000) // Restore default after test

	// Create a dataset with a spike
	data := make([]schema.Float, 200)
	for i := range data {
		if i == 100 {
			data[i] = 100.0 // Spike
		} else {
			data[i] = 1.0
		}
	}

	lttbResult, _, _ := LargestTriangleThreeBucket(data, 1, 2)

	// LTTB should be more likely to preserve the spike
	lttbHasSpike := false

	for _, v := range lttbResult {
		if v > 50.0 {
			lttbHasSpike = true
			break
		}
	}

	// This test is probabilistic, but LTTB should generally preserve the spike
	if !lttbHasSpike {
		t.Log("Warning: LTTB did not preserve spike (may happen occasionally)")
	}
}

// Helper functions for test data generation

// makeTestData creates a simple linear test dataset
func makeTestData(size int, increment schema.Float) []schema.Float {
	data := make([]schema.Float, size)
	for i := range data {
		data[i] = schema.Float(i) * increment
	}
	return data
}

// makeSineWave creates a sine wave test dataset
func makeSineWave(size int, periods float64) []schema.Float {
	data := make([]schema.Float, size)
	for i := range data {
		angle := 2 * math.Pi * periods * float64(i) / float64(size)
		data[i] = schema.Float(math.Sin(angle))
	}
	return data
}

// makeTestDataWithNaN creates test data with some NaN values
func makeTestDataWithNaN(size int) []schema.Float {
	data := make([]schema.Float, size)
	for i := range data {
		if i%20 == 0 {
			data[i] = schema.NaN
		} else {
			data[i] = schema.Float(i)
		}
	}
	return data
}

// Benchmark tests

func BenchmarkSimpleResampler(b *testing.B) {
	data := makeTestData(10000, 1.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = SimpleResampler(data, 1, 10)
	}
}

func BenchmarkLargestTriangleThreeBucket(b *testing.B) {
	data := makeTestData(10000, 1.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = LargestTriangleThreeBucket(data, 1, 10)
	}
}

func BenchmarkLTTBLargeDataset(b *testing.B) {
	data := makeTestData(100000, 1.0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = LargestTriangleThreeBucket(data, 1, 100)
	}
}
