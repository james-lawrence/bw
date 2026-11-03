package bytesx

import "fmt"

// Human formats b using base 2 units with a single decimal, e.g. "1.5 MiB".
func Human(b uint64) string {
	div := uint64(1)
	suffix := "B"

	switch {
	case b >= EiB:
		div, suffix = EiB, "EiB"
	case b >= PiB:
		div, suffix = PiB, "PiB"
	case b >= TiB:
		div, suffix = TiB, "TiB"
	case b >= GiB:
		div, suffix = GiB, "GiB"
	case b >= MiB:
		div, suffix = MiB, "MiB"
	case b >= KiB:
		div, suffix = KiB, "KiB"
	}

	return fmt.Sprintf("%.1f %s", float64(b)/float64(div), suffix)
}
