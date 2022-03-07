// Code generated by 'option-gen'. DO NOT EDIT.

package kube

import (
	prompt "github.com/RafaySystems/prompt/pkg/prompt"
)

var proxyOptions = []prompt.Suggest{
	prompt.Suggest{Text: "--accept-hosts", Description: "Regular expression for hosts that the proxy should accept."},
	prompt.Suggest{Text: "--accept-paths", Description: "Regular expression for paths that the proxy should accept."},
	prompt.Suggest{Text: "--address", Description: "The IP address on which to serve on."},
	prompt.Suggest{Text: "--api-prefix", Description: "Prefix to serve the proxied API under."},
	prompt.Suggest{Text: "--disable-filter", Description: "If true, disable request filtering in the proxy. This is dangerous, and can leave you vulnerable to XSRF attacks, when used with an accessible port."},
	prompt.Suggest{Text: "--keepalive", Description: "keepalive specifies the keep-alive period for an active network connection. Set to 0 to disable keepalive."},
	prompt.Suggest{Text: "-p", Description: "The port on which to run the proxy. Set to 0 to pick a random port."},
	prompt.Suggest{Text: "--port", Description: "The port on which to run the proxy. Set to 0 to pick a random port."},
	prompt.Suggest{Text: "--reject-methods", Description: "Regular expression for HTTP methods that the proxy should reject (example --reject-methods='POST,PUT,PATCH')."},
	prompt.Suggest{Text: "--reject-paths", Description: "Regular expression for paths that the proxy should reject. Paths specified here will be rejected even accepted by --accept-paths."},
	prompt.Suggest{Text: "-u", Description: "Unix socket on which to run the proxy."},
	prompt.Suggest{Text: "--unix-socket", Description: "Unix socket on which to run the proxy."},
	prompt.Suggest{Text: "-w", Description: "Also serve static files from the given directory under the specified prefix."},
	prompt.Suggest{Text: "--www", Description: "Also serve static files from the given directory under the specified prefix."},
	prompt.Suggest{Text: "-P", Description: "Prefix to serve static files under, if static file directory is specified."},
	prompt.Suggest{Text: "--www-prefix", Description: "Prefix to serve static files under, if static file directory is specified."},
}
