package suite

import (
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
)

func TestScenarioTimingDefaults(t *testing.T) {
	t.Setenv("KDEN_CONFORMANCE_TIMEOUT", "")
	t.Setenv("KDEN_CONFORMANCE_OBSERVATION_TIMEOUT", "")
	timing, err := loadScenarioTiming()
	if err != nil || timing.operation != time.Minute || timing.observation != 10*time.Second {
		t.Fatalf("unexpected defaults: %+v, %v", timing, err)
	}
}

func TestScenarioTimingOverrides(t *testing.T) {
	t.Setenv("KDEN_CONFORMANCE_TIMEOUT", "2s")
	t.Setenv("KDEN_CONFORMANCE_OBSERVATION_TIMEOUT", "3s")
	timing, err := loadScenarioTiming()
	if err != nil || timing.operation != 2*time.Second || timing.observation != 3*time.Second {
		t.Fatalf("unexpected timeout configuration: %+v, %v", timing, err)
	}
	if timing.statusSpecTimeout() != ginkgo.SpecTimeout(4*time.Second) {
		t.Fatal("status spec budget must allow setup and a status wait")
	}
	if timing.observationSpecTimeout() != ginkgo.SpecTimeout(7*time.Second) {
		t.Fatal("ownership spec budget must additionally allow the observation window")
	}
}

func TestScenarioTimingRejectsInvalidDurations(t *testing.T) {
	for _, name := range []string{"KDEN_CONFORMANCE_TIMEOUT", "KDEN_CONFORMANCE_OBSERVATION_TIMEOUT"} {
		for _, value := range []string{"invalid", "0s", "-1s"} {
			t.Run(name+"/"+value, func(t *testing.T) {
				t.Setenv("KDEN_CONFORMANCE_TIMEOUT", "1s")
				t.Setenv("KDEN_CONFORMANCE_OBSERVATION_TIMEOUT", "1s")
				t.Setenv(name, value)
				timing, err := loadScenarioTiming()
				if err == nil {
					t.Fatalf("accepted invalid duration %q", value)
				}
				// The tree must still receive valid decorators so Ginkgo can
				// report the configuration failure in the setup hook.
				if timing.operation <= 0 || timing.observation <= 0 {
					t.Fatal("invalid configuration produced an invalid Ginkgo timeout decorator")
				}
			})
		}
	}
}
