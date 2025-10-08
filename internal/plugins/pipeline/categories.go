package pipeline

import (
	pkg "github.com/peteski22/plugins-demo/pkg/contract/plugin"
)

// categoryProps maps each category to its execution properties.
// The pipeline enforces these constraints during request/response processing.
var categoryProps = map[pkg.Category]pkg.CategoryProperties{
	pkg.CategoryAuthN:         {Mode: pkg.ExecSerial, CanReject: true, CanModify: false},
	pkg.CategoryAuthZ:         {Mode: pkg.ExecSerial, CanReject: true, CanModify: false},
	pkg.CategoryRateLimiting:  {Mode: pkg.ExecSerial, CanReject: true, CanModify: false},
	pkg.CategoryValidation:    {Mode: pkg.ExecSerial, CanReject: true, CanModify: false},
	pkg.CategoryContent:       {Mode: pkg.ExecSerial, CanReject: true, CanModify: true},
	pkg.CategoryObservability: {Mode: pkg.ExecParallel, CanReject: false, CanModify: false},
}

// OrderedCategories defines the pipeline execution order.
// Categories execute in this sequence for each request/response.
var OrderedCategories = []pkg.Category{
	pkg.CategoryObservability,
	pkg.CategoryAuthN,
	pkg.CategoryAuthZ,
	pkg.CategoryRateLimiting,
	pkg.CategoryValidation,
	pkg.CategoryContent,
}

// PropsForCategory returns properties for a category. Unknown categories fall back
// to a conservative default (serial, non-modifying, non-rejecting).
func PropsForCategory(c pkg.Category) pkg.CategoryProperties {
	if p, ok := categoryProps[c]; ok {
		return p
	}

	// TODO: Use standard/default property set.
	return pkg.CategoryProperties{Mode: pkg.ExecSerial, CanReject: false, CanModify: false}
}
