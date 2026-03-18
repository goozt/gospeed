package results

import "testing"

func TestGradeLatency(t *testing.T) {
	tests := []struct {
		ms   float64
		want Grade
	}{
		{5, GradeA},
		{19.9, GradeA},
		{20, GradeB},
		{49.9, GradeB},
		{50, GradeC},
		{99.9, GradeC},
		{100, GradeD},
		{199.9, GradeD},
		{200, GradeF},
		{500, GradeF},
	}
	for _, tt := range tests {
		got := GradeLatency(tt.ms)
		if got != tt.want {
			t.Errorf("GradeLatency(%.1f) = %s, want %s", tt.ms, got, tt.want)
		}
	}
}

func TestGradeLoss(t *testing.T) {
	tests := []struct {
		pct  float64
		want Grade
	}{
		{0, GradeA},
		{0.09, GradeA},
		{0.1, GradeB},
		{0.5, GradeC},
		{1.0, GradeD},
		{2.5, GradeF},
	}
	for _, tt := range tests {
		got := GradeLoss(tt.pct)
		if got != tt.want {
			t.Errorf("GradeLoss(%.2f) = %s, want %s", tt.pct, got, tt.want)
		}
	}
}

func TestGradeJitter(t *testing.T) {
	tests := []struct {
		ms   float64
		want Grade
	}{
		{1, GradeA},
		{5, GradeB},
		{10, GradeC},
		{20, GradeD},
		{50, GradeF},
	}
	for _, tt := range tests {
		got := GradeJitter(tt.ms)
		if got != tt.want {
			t.Errorf("GradeJitter(%.1f) = %s, want %s", tt.ms, got, tt.want)
		}
	}
}

func TestGradeBufferbloat(t *testing.T) {
	tests := []struct {
		rpm  float64
		want Grade
	}{
		{500, GradeA},
		{400, GradeA},
		{200, GradeB},
		{100, GradeC},
		{50, GradeD},
		{10, GradeF},
	}
	for _, tt := range tests {
		got := GradeBufferbloat(tt.rpm)
		if got != tt.want {
			t.Errorf("GradeBufferbloat(%.0f) = %s, want %s", tt.rpm, got, tt.want)
		}
	}
}

func TestComputeOverallGrade(t *testing.T) {
	results := []TestResult{
		{Grade: GradeA},
		{Grade: GradeB},
		{Grade: GradeC},
	}
	got := ComputeOverallGrade(results)
	if got != GradeC {
		t.Errorf("overall = %s, want C", got)
	}
}
