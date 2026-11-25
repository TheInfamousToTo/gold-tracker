package report

import "testing"

func TestWeekDrop(t *testing.T) {
	s := []point{
		{"2026-11-01", 60}, // outside the week: ignored
		{"2026-11-10", 50},
		{"2026-11-12", 52},
		{"2026-11-17", 49},
	}
	d, ok := weekDrop(s)
	if !ok || d.Peak.Price != 52 || d.Latest.Price != 49 {
		t.Fatalf("drop = %+v, %v", d, ok)
	}
	if d.Pct > -5.7 || d.Pct < -5.8 {
		t.Fatalf("pct = %v, want about -5.77", d.Pct)
	}
}

func TestShouldAlert(t *testing.T) {
	d := drop{Peak: point{"2026-11-12", 52}, Latest: point{"2026-11-17", 49}, Pct: -5.77}
	cases := []struct {
		name      string
		threshold float64
		lastDate  string
		lastPrice float64
		alerted   bool
		want      bool
	}{
		{"small drop", 8, "", 0, false, false},
		{"first alert", 5, "", 0, false, true},
		{"already sent today", 5, "2026-11-17", 49, true, false},
		{"same slide, day later", 5, "2026-11-16", 50, true, false},
		{"fell another 5%", 5, "2026-11-15", 52, true, true},
		{"a week on, still down", 5, "2026-11-10", 49.5, true, true},
	}
	for _, c := range cases {
		if got := shouldAlert(d, c.threshold, c.lastDate, c.lastPrice, c.alerted); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
