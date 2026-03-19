package results

// Grade represents a quality rating from A to F.
type Grade string

const (
	GradeA Grade = "A"
	GradeB Grade = "B"
	GradeC Grade = "C"
	GradeD Grade = "D"
	GradeF Grade = "F"
)

// GradeLatency returns a grade based on average latency in ms.
func GradeLatency(avgMs float64) Grade {
	switch {
	case avgMs < 20:
		return GradeA
	case avgMs < 50:
		return GradeB
	case avgMs < 100:
		return GradeC
	case avgMs < 200:
		return GradeD
	default:
		return GradeF
	}
}

// GradeConnect returns a grade based on average TCP connect time in ms.
func GradeConnect(avgMs float64) Grade {
	switch {
	case avgMs < 50:
		return GradeA
	case avgMs < 100:
		return GradeB
	case avgMs < 200:
		return GradeC
	case avgMs < 500:
		return GradeD
	default:
		return GradeF
	}
}

// GradeDNS returns a grade based on average DNS resolution time in ms.
func GradeDNS(avgMs float64) Grade {
	switch {
	case avgMs < 10:
		return GradeA
	case avgMs < 30:
		return GradeB
	case avgMs < 75:
		return GradeC
	case avgMs < 200:
		return GradeD
	default:
		return GradeF
	}
}

// GradeMTU returns a grade based on path MTU size in bytes.
func GradeMTU(mtu int) Grade {
	switch {
	case mtu >= 1500:
		return GradeA
	case mtu >= 1400:
		return GradeB
	case mtu >= 1200:
		return GradeC
	case mtu >= 576:
		return GradeD
	default:
		return GradeF
	}
}

// GradeLoss returns a grade based on packet loss percentage.
func GradeLoss(pct float64) Grade {
	switch {
	case pct < 0.1:
		return GradeA
	case pct < 0.5:
		return GradeB
	case pct < 1.0:
		return GradeC
	case pct < 2.5:
		return GradeD
	default:
		return GradeF
	}
}

// GradeJitter returns a grade based on average jitter in ms.
func GradeJitter(avgMs float64) Grade {
	switch {
	case avgMs < 5:
		return GradeA
	case avgMs < 10:
		return GradeB
	case avgMs < 20:
		return GradeC
	case avgMs < 50:
		return GradeD
	default:
		return GradeF
	}
}

// GradeBufferbloat returns a grade based on RPM (round-trips per minute).
func GradeBufferbloat(rpm float64) Grade {
	switch {
	case rpm >= 400:
		return GradeA
	case rpm >= 200:
		return GradeB
	case rpm >= 100:
		return GradeC
	case rpm >= 50:
		return GradeD
	default:
		return GradeF
	}
}

// GradeThroughput returns a grade based on bits per second.
func GradeThroughput(bps float64) Grade {
	mbps := bps / 1_000_000
	switch {
	case mbps >= 100:
		return GradeA
	case mbps >= 50:
		return GradeB
	case mbps >= 25:
		return GradeC
	case mbps >= 10:
		return GradeD
	default:
		return GradeF
	}
}
