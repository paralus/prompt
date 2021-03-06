// Code generated by 'option-gen'. DO NOT EDIT.

package kube

import (
	prompt "github.com/paralus/prompt/pkg/prompt"
)

var attachOptions = []prompt.Suggest{
	{Text: "-c", Description: "Container name. If omitted, the first container in the pod will be chosen"},
	{Text: "--container", Description: "Container name. If omitted, the first container in the pod will be chosen"},
	{Text: "--pod-running-timeout", Description: "The length of time (like 5s, 2m, or 3h, higher than zero) to wait until at least one pod is running"},
	{Text: "-i", Description: "Pass stdin to the container"},
	{Text: "--stdin", Description: "Pass stdin to the container"},
	{Text: "-t", Description: "Stdin is a TTY"},
	{Text: "--tty", Description: "Stdin is a TTY"},
}
