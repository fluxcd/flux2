package sync

import "time"

type Options struct {
	Interval   time.Duration
	URL        string
	Name       string
	Namespace  string
	Branch     string
	TargetPath string
}

func MakeDefaultOptions() Options {
	return Options{
		Interval:   1 * time.Minute,
		URL:        "",
		Name:       "gotk-system",
		Namespace:  "gotk-system",
		Branch:     "main",
		TargetPath: "",
	}
}
