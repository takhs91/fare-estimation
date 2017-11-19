package main

import (
    "testing"
    "time"
)

// https://gist.github.com/cevaris/bc331cbe970b03816c6b
var EPSILON  = 0.00000001
func floatEquals(a, b float64) bool {
    if ((a - b) < EPSILON && (b - a) < EPSILON) {
        return true
    }
    return false
}

func TestStringArrayToRecordt(t *testing.T) {
    cases := []struct {
        in []string
        want Record
        err bool
    }{
        {[]string{"1", "37.2", "23.2", "1405588381"}, Record{1, 37.2, 23.2, 1405588381}, false},
        {[]string{"9", "37.6", "23.3", "1405588400"}, Record{9, 37.6, 23.3, 1405588400}, false},
        // Malformed, float id
        {[]string{"9.3", "37.6", "23.3", "1405588400"}, Record{}, true},
        // Malformed1, float timestamp
        {[]string{"9", "37.6", "23.3", "1405588400.23"}, Record{}, true},
    }
    for _, c := range cases {
        got, err := StringArrayToRecord(c.in)
        if got != c.want {
            t.Errorf("TestStringArrayToRecordt(%q) == %q, want %q", c.in, got, c.want)
        }
        if c.err {
            if err == nil{
                t.Errorf("Error should't be nil for case(%q)", c.in)
            }
        } else {
            if err != nil{
                t.Errorf("Error should be nil for case(%q)", c.in)
            }
        }
    }
}


func TestEstimateSegmentFare(t *testing.T) {
    t1 := time.Date(2014, time.August, 15, 23, 59, 40, 0, time.Local)
    t2 := time.Date(2014, time.August, 15, 23, 59, 50, 0, time.Local)
    t6 := time.Date(2014, time.August, 16, 00, 00, 00, 0, time.Local)
    t3 := time.Date(2014, time.August, 16, 00, 00, 10, 0, time.Local)
    t4 := time.Date(2014, time.August, 16, 00, 00, 20, 0, time.Local)
    t5 := time.Date(2014, time.August, 16, 4, 59, 50, 0, time.Local)
    t7 := time.Date(2014, time.August, 16, 5, 0, 0, 0, time.Local)
    t8 := time.Date(2014, time.August, 16, 5, 0, 10, 0, time.Local)
    t9 := time.Date(2014, time.August, 16, 5, 0, 20, 0, time.Local)
    cases := []struct {
        in *Segment
        want float64
    }{
        // Test for Idle
        {&Segment{U: 0.0, DeltaS: 0.0, DeltaT: time.Second * 10, T1: t1, T2: t2}, IdleChargePerHour * (time.Second * 10).Hours()},
        // Test Normal Charge
        {&Segment{U: 20.0, DeltaS: 0.056, DeltaT: time.Second * 10, T1: t1, T2: t2}, NormalChargePerKilometer * 0.056},
        {&Segment{U: 20.0, DeltaS: 0.056, DeltaT: time.Second * 10, T1: t8, T2: t9}, NormalChargePerKilometer * 0.056},
        {&Segment{U: 20.0, DeltaS: 0.056, DeltaT: time.Second * 10, T1: t7, T2: t8}, NormalChargePerKilometer * 0.056},
        // Test NightChargePerKilometer
        {&Segment{U: 20.0, DeltaS: 0.056, DeltaT: time.Second * 10, T1: t3, T2: t4}, NightChargePerKilometer * 0.056},
        {&Segment{U: 20.0, DeltaS: 0.056, DeltaT: time.Second * 10, T1: t5, T2: t7}, NightChargePerKilometer * 0.056},
        {&Segment{U: 20.0, DeltaS: 0.056, DeltaT: time.Second * 10, T1: t6, T2: t3}, NightChargePerKilometer * 0.056},
        // Test 10s before midnight and 10s after midnight
        {&Segment{U: 20.0, DeltaS: 0.100, DeltaT: time.Second * 20, T1: t2, T2: t3}, NormalChargePerKilometer * 0.100 * 0.5 + NightChargePerKilometer * 0.100 * 0.5},
        // Test 10s before five and 10s after five
        {&Segment{U: 20.0, DeltaS: 0.100, DeltaT: time.Second * 20, T1: t5, T2: t8}, NormalChargePerKilometer * 0.100 * 0.5 + NightChargePerKilometer * 0.100 * 0.5},
        //Test 10s before midnight and 20s after
        {&Segment{U: 20.0, DeltaS: 0.100, DeltaT: time.Second * 30, T1: t2, T2: t4}, NormalChargePerKilometer * 0.100 * (1.0/3.0) + NightChargePerKilometer * 0.100 * (2.0/3.0)},
    }
    for _, c := range cases {
        got := EstimateSegmentFare(c.in)
        if !floatEquals(got, c.want) {
            t.Errorf("EstimateSegmentFare(%q) == %q, want %q", c.in, got, c.want)
        }
    }
}
