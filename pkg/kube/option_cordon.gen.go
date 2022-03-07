// Code generated by 'option-gen'. DO NOT EDIT.

package kube

import (
	prompt "github.com/RafaySystems/prompt/pkg/prompt"
)

var cordonOptions = []prompt.Suggest{
	prompt.Suggest{Text: "--dry-run", Description: "If true, only print the object that would be sent, without sending it."},
	prompt.Suggest{Text: "-l", Description: "Selector (label query) to filter on"},
	prompt.Suggest{Text: "--selector", Description: "Selector (label query) to filter on"},
}
