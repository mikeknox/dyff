package dyff

import (
	"regexp"

	"github.com/gonvenience/ytbx"
	"gopkg.in/yaml.v3"
)

func (r Report) filter(hasPath func(*ytbx.Path) bool) (result Report) {
	result = Report{
		From: r.From,
		To:   r.To,
	}

	includeDiff := true
	for _, diff := range r.Diffs {
		if !hasPath(diff.Path) {
			includeDiff = false
		}
		for _, diffDetail := range diff.Details {
			for _, diffNode := range []*yaml.Node{diffDetail.From, diffDetail.To} {
				if diffNode != nil {
					subPaths, err := ytbx.ListPathsInNode(diffNode)
					if err == nil {
						for _, subPath := range subPaths {
							testPath := ytbx.AppendPath(*diff.Path, subPath)
							if !hasPath(&testPath) {
								includeDiff = false
							}
						}
					}
				}
			}
		}
		if includeDiff {
			result.Diffs = append(result.Diffs, diff)
		}
	}

	return result
}

// Filter accepts YAML paths as input and returns a new report with differences for those paths only
func (r Report) Filter(paths ...string) (result Report) {
	if len(paths) == 0 {
		return r
	}

	return r.filter(func(filterPath *ytbx.Path) bool {
		for _, pathString := range paths {
			path, err := ytbx.ParsePathStringUnsafe(pathString)
			if err == nil && filterPath != nil && path.String() == filterPath.String() {
				return true
			}
		}

		return false
	})
}

// Exclude accepts YAML paths as input and returns a new report with differences without those paths
func (r Report) Exclude(paths ...string) (result Report) {
	if len(paths) == 0 {
		return r
	}

	return r.filter(func(filterPath *ytbx.Path) bool {
		for _, pathString := range paths {
			path, err := ytbx.ParsePathStringUnsafe(pathString)
			if err == nil && filterPath != nil && path.String() == filterPath.String() {
				return false
			}
		}

		return true
	})
}

// FilterRegexp accepts regular expressions as input and returns a new report with differences for matching those patterns
func (r Report) FilterRegexp(pattern ...string) (result Report) {
	if len(pattern) == 0 {
		return r
	}

	regexps := make([]*regexp.Regexp, len(pattern))
	for i := range pattern {
		regexps[i] = regexp.MustCompile(pattern[i])
	}

	return r.filter(func(filterPath *ytbx.Path) bool {
		for _, regexp := range regexps {
			if filterPath != nil && regexp.MatchString(filterPath.String()) {
				return true
			}
		}
		return false
	})
}

// ExcludeRegexp accepts regular expressions as input and returns a new report with differences for not matching those patterns
func (r Report) ExcludeRegexp(pattern ...string) (result Report) {
	if len(pattern) == 0 {
		return r
	}

	regexps := make([]*regexp.Regexp, len(pattern))
	for i := range pattern {
		regexps[i] = regexp.MustCompile(pattern[i])
	}

	return r.filter(func(filterPath *ytbx.Path) bool {
		for _, regexp := range regexps {
			if filterPath != nil && regexp.MatchString(filterPath.String()) {
				return false
			}
		}
		return true
	})
}

func (r Report) IgnoreValueChanges() (result Report) {
	result = Report{
		From: r.From,
		To:   r.To,
	}

	for _, diff := range r.Diffs {
		var hasValChange = false
		for _, detail := range diff.Details {
			if detail.Kind == MODIFICATION {
				hasValChange = true
				break
			}
		}

		if !hasValChange {
			result.Diffs = append(result.Diffs, diff)
		}
	}

	return result
}
