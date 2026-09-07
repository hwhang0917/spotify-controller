package source

import "testing"

func TestYearOf(t *testing.T) {
	for in, want := range map[string]int{"2019-04-01T10:00:00Z": 2019, "1981-12": 1981, "2000": 2000, "": 0, "abcd-01": 0, "19": 0} {
		if got := YearOf(in); got != want {
			t.Errorf("YearOf(%q) = %d, want %d", in, got, want)
		}
	}
}
