// Code generated by 'option-gen'. DO NOT EDIT.

package kube

import (
	prompt "github.com/RafaySystems/prompt/pkg/prompt"
)

var configViewOptions = []prompt.Suggest{
	prompt.Suggest{Text: "--allow-missing-template-keys", Description: "If true, ignore any errors in templates when a field or map key is missing in the template. Only applies to golang and jsonpath output formats."},
	prompt.Suggest{Text: "--flatten", Description: "Flatten the resulting kubeconfig file into self-contained output (useful for creating portable kubeconfig files)"},
	prompt.Suggest{Text: "--merge", Description: "Merge the full hierarchy of kubeconfig files"},
	prompt.Suggest{Text: "--minify", Description: "Remove all information not used by current-context from the output"},
	prompt.Suggest{Text: "-o", Description: "Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file."},
	prompt.Suggest{Text: "--output", Description: "Output format. One of: json|yaml|name|go-template|go-template-file|template|templatefile|jsonpath|jsonpath-file."},
	prompt.Suggest{Text: "--raw", Description: "Display raw byte data"},
	prompt.Suggest{Text: "--template", Description: "Template string or path to template file to use when -o=go-template, -o=go-template-file. The template format is golang templates [http://golang.org/pkg/text/template/#pkg-overview]."},
}