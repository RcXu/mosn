/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cluster

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

func Test(t *testing.T) {
	A := &mockHost{name: "A", w: 4}
	B := &mockHost{name: "B", w: 2}
	C := &mockHost{name: "C", w: 3}
	D := &mockHost{name: "D", w: 1}

	edfScheduler := newEdfScheduler(4)
	edfScheduler.Add(A, float64(A.w))
	edfScheduler.Add(B, float64(B.w))
	edfScheduler.Add(C, float64(C.w))
	edfScheduler.Add(D, float64(D.w))
	weightFunc := func(item WeightItem) float64 {
		return float64(item.Weight())
	}
	ele := edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, A, ele)
	ele = edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, C, ele)
	ele = edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, B, ele)
	ele = edfScheduler.NextAndPush(weightFunc)
	assert.Equal(t, A, ele)

}

func TestEdfFixedWeight(t *testing.T) {
	if edfFixedWeight(0) != float64(v2.MinHostWeight) {
		t.Fatalf("Except %f but %f", float64(v2.MinHostWeight), edfFixedWeight(0))
	}
	if edfFixedWeight(math.MaxFloat64) != float64(v2.MaxHostWeight) {
		t.Fatalf("Except %f but %f", float64(v2.MaxHostWeight), edfFixedWeight(math.MaxFloat64))
	}
	if edfFixedWeight(10.0) != 10.0 {
		t.Fatalf("Except %f but %f", 10.0, edfFixedWeight(10.0))
	}
}

func mockHostList(count int, name string) []types.Host {
	hosts := make([]types.Host, 0, count)
	for i := 0; i < count; i++ {
		hosts = append(hosts, &mockHost{
			name: "A" + strconv.Itoa(i),
			w:    uint32(i + 1),
		})
	}
	return hosts
}

func Benchmark_edfSchduler_NextAndPush(b *testing.B) {
	type fields struct {
		hostCount int
		hostName  string
	}
	type args struct {
		weightFunc func(item WeightItem) float64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
	}{
		{
			name: "10-hosts",
			fields: fields{
				hostCount: 10,
				hostName:  "bench-host",
			},
			args: args{
				weightFunc: func(item WeightItem) float64 {
					return float64(item.Weight())
				},
			},
		},
		{
			name: "100-hosts",
			fields: fields{
				hostCount: 100,
				hostName:  "bench-host",
			},
			args: args{
				weightFunc: func(item WeightItem) float64 {
					return float64(item.Weight())
				},
			},
		},
		{
			name: "1000-hosts",
			fields: fields{
				hostCount: 1000,
				hostName:  "bench-host",
			},
			args: args{
				weightFunc: func(item WeightItem) float64 {
					return float64(item.Weight())
				},
			},
		},
	}
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			edf := newEdfScheduler(tt.fields.hostCount)
			hosts := mockHostList(tt.fields.hostCount, tt.fields.hostName)
			for _, h := range hosts {
				edf.Add(h, float64(h.Weight()))
			}
			b.StartTimer()
			for i := 0; i < b.N; i++ {
				edf.NextAndPush(tt.args.weightFunc)
			}
			b.StopTimer()
		})
	}
}

func TestEdfSchedulerDistribution(t *testing.T) {
	var weights []uint32
	totalWeights := uint32(0)

	rnd := func(low, high int) int {
		return rand.Intn(high-low) + low
	}

	checkDistribution := func(seq []string) {
		dist := make(map[string]int)
		for _, s := range seq {
			dist[s]++
		}
		for i, w := range weights {
			d := dist[fmt.Sprintf("host-%d", i)]
			assert.Equal(t, uint32(d), w)
		}
	}

	// number of hosts in [2*MaxHostWeight, 4*MaxHostWeight) to make sure
	// always have two hosts with same weight
	for i := rnd(2*int(v2.MaxHostWeight), 4*int(v2.MaxHostWeight)); i >= 0; i-- {
		w := uint32(rnd(int(v2.MinHostWeight), int(v2.MaxHostWeight)))
		weights = append(weights, w)
		totalWeights += w
	}

	scheduler := newEdfScheduler(len(weights))
	for i, w := range weights {
		scheduler.Add(&mockHost{name: fmt.Sprintf("host-%d", i), w: w}, float64(w))
	}

	for i := 0; i < 128; i++ {
		seq := make([]string, 0)
		for i := uint32(0); i < totalWeights; i++ {
			h := scheduler.NextAndPush(func(item WeightItem) float64 {
				return float64(item.Weight())
			}).(*mockHost)
			seq = append(seq, h.name)
		}
		checkDistribution(seq)
	}
}
