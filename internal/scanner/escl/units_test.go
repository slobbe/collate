package escl

import "testing"

func TestESCLUnitMicrometreRoundTrip(t *testing.T) {
	for _, micrometres := range []int{1, 85, 1_000, 210_000, 297_000} {
		units := toESCLUnits(micrometres)
		got := toMicrometres(units)
		if difference := got - micrometres; difference < -42 || difference > 42 {
			t.Fatalf("round trip for %d µm through %d units = %d µm", micrometres, units, got)
		}
	}
}

func TestToMicrometresRoundsByDivisorHalf(t *testing.T) {
	if got, want := toMicrometres(1), 85; got != want {
		t.Fatalf("toMicrometres(1) = %d, want %d", got, want)
	}
}
