package sql

import "fmt"

// CheckSites validates backend parameter sites against the declared
// total: every site number must fall within 1..Total, and sites must
// arrive in nondecreasing source order. It returns the set of numbers
// present. Backends report only genuine sites; text resembling parameters
// inside quotes or comments never reaches this check.
func CheckSites(name string, sites []ParamSite, total int) (map[int]bool, error) {
	present := map[int]bool{}
	last := 0
	for _, site := range sites {
		if site.Start < last {
			return nil, fmt.Errorf("sql descriptor %q: parameter sites out of order", name)
		}
		last = site.Start
		if site.Number < 1 || site.Number > total {
			return nil, fmt.Errorf("sql descriptor %q: %s has no declared parameter", name, site.Ref)
		}
		present[site.Number] = true
	}
	return present, nil
}

// CheckCoverage requires every declared number to appear at least once.
// The row-limit number reports its own absence; application numbers name
// their declared parameter. Numbers render in the dialect's spelling.
func CheckCoverage(name string, dialect Dialect, parameters []string, present map[int]bool, total, limit int, rowReturning bool) error {
	for n := 1; n <= total; n++ {
		if !present[n] {
			if rowReturning && n == total {
				return fmt.Errorf("sql descriptor %q: row-limit parameter %s is not used", name, limitRef(dialect, n))
			}
			return fmt.Errorf("sql descriptor %q: parameter %q (%s) is not used", name, parameters[n-1], limitRef(dialect, n))
		}
	}
	return nil
}

// CheckSiteNames requires named SQLite sites to spell their declared
// parameter: a :name/@name/$name site binding an application number must
// carry that number's declared name exactly. Bare and ?NNN sites have no
// name to check, and the row-limit site's name is free since no declared
// parameter corresponds to it. Sigils never distinguish names. Ranges and
// coverage already passed, so out-of-range numbers cannot occur here.
func CheckSiteNames(name string, parameters []string, sites []ParamSite, limit int) error {
	for _, site := range sites {
		if len(site.Ref) == 0 || (site.Ref[0] != ':' && site.Ref[0] != '@' && site.Ref[0] != '$') {
			continue
		}
		if limit != 0 && site.Number == limit {
			continue
		}
		if site.Number < 1 || site.Number > len(parameters) {
			continue
		}
		if want := parameters[site.Number-1]; site.Ref[1:] != want {
			return fmt.Errorf("sql descriptor %q: site %s does not match declared parameter %q", name, site.Ref, want)
		}
	}
	return nil
}

// TileSegments tiles the exact statement bytes into static text and
// parameter sites. Sites arrive in source order and never overlap;
// literal gaps become Text segments, including trivia before the first
// site and after the last.
func TileSegments(statement string, sites []ParamSite) []Segment {
	var out []Segment
	cursor := 0
	for _, site := range sites {
		if literal := statement[cursor:site.Start]; literal != "" {
			out = append(out, Segment{Text: literal})
		}
		out = append(out, Segment{Param: site.Number})
		cursor = site.End
	}
	if literal := statement[cursor:]; literal != "" {
		out = append(out, Segment{Text: literal})
	}
	return out
}
