package requestinput

import "strings"

// friendlyError is a human-readable explanation of a request failure. The raw
// Go error is still kept on ResponsePanel.Err and is only shown in Raw mode.
type friendlyError struct {
	Title       string
	Cause       string
	Suggestions []string
}

func classifyError(raw string) friendlyError {
	lower := strings.ToLower(strings.TrimSpace(raw))

	switch {
	case lower == "":
		return friendlyError{
			Title: "Request failed",
			Cause: "An unknown error occurred.",
		}
	case strings.Contains(lower, "no such host"),
		strings.Contains(lower, "server misbehaving"),
		strings.Contains(lower, "temporary failure in name resolution"),
		strings.Contains(lower, "dns"):
		return friendlyError{
			Title: "Host not found",
			Cause: "The domain name could not be resolved.",
			Suggestions: []string{
				"Check the hostname spelling in the URL",
				"Make sure a VPN/proxy is not blocking DNS",
			},
		}
	case strings.Contains(lower, "connection refused"):
		return friendlyError{
			Title: "Connection refused",
			Cause: "The server is not accepting connections on that host/port.",
			Suggestions: []string{
				"Make sure the server is running",
				"Check the port and host in the URL",
			},
		}
	case strings.Contains(lower, "certificate"),
		strings.Contains(lower, "x509"),
		strings.Contains(lower, "tls"),
		strings.Contains(lower, "handshake"):
		return friendlyError{
			Title: "TLS verification failed",
			Cause: "The secure connection could not be verified.",
			Suggestions: []string{
				"Use http:// for local development",
				"Or add the CA to the system trust store",
			},
		}
	case strings.Contains(lower, "timeout"),
		strings.Contains(lower, "deadline exceeded"),
		strings.Contains(lower, "timed out"):
		return friendlyError{
			Title: "Request timed out",
			Cause: "The server did not respond in time.",
			Suggestions: []string{
				"Retry the request",
				"Check whether the server is down or slow",
			},
		}
	case strings.Contains(lower, "credentials blocked"):
		return friendlyError{
			Title: "Request blocked by Scope",
			Cause: "Credentials were sent to an origin that is not in the allowlist.",
			Suggestions: []string{
				"Add the exact request origin to .termapi/global.yaml",
				"Or remove sensitive headers/auth for this request",
			},
		}
	case strings.Contains(lower, "redirect blocked"):
		return friendlyError{
			Title: "Redirect blocked",
			Cause: "The redirect target is outside the Scope allowlist.",
			Suggestions: []string{
				"Allow the redirect origin in .termapi/global.yaml",
				"Or disable redirects for this request",
			},
		}
	case strings.Contains(lower, "stopped after 10 redirects"):
		return friendlyError{
			Title: "Too many redirects",
			Cause: "The request was redirected more than 10 times.",
			Suggestions: []string{
				"Check the URL for a redirect loop",
				"Point the request directly at the final URL",
			},
		}
	case strings.Contains(lower, "url is empty"),
		strings.Contains(lower, "invalid url"),
		strings.Contains(lower, "missing host"):
		return friendlyError{
			Title: "Invalid URL",
			Cause: "The request URL is empty or not valid.",
			Suggestions: []string{
				"Include a scheme, for example https://api.example.com",
				"Check for stray spaces or missing host",
			},
		}
	case strings.Contains(lower, "invalid header line"):
		return friendlyError{
			Title: "Invalid header",
			Cause: "A header name or value contains invalid characters.",
			Suggestions: []string{
				"Check the Headers tab for empty or malformed rows",
			},
		}
	case strings.Contains(lower, "invalid param"):
		return friendlyError{
			Title: "Invalid parameter",
			Cause: "A query parameter has a value but no key.",
			Suggestions: []string{
				"Fill in the KEY column or remove the row in Params",
			},
		}
	case strings.Contains(lower, "multipart file"),
		strings.Contains(lower, "binary body"):
		return friendlyError{
			Title: "Cannot read body file",
			Cause: "The file used by the request body could not be read.",
			Suggestions: []string{
				"Check that the file path exists and is readable",
			},
		}
	case strings.Contains(lower, "read response body"):
		return friendlyError{
			Title: "Failed to read response",
			Cause: "The connection was interrupted while receiving the response.",
			Suggestions: []string{
				"Retry the request",
				"Check the server logs for errors",
			},
		}
	default:
		return friendlyError{
			Title: "Request failed",
			Cause: "An unexpected error occurred while sending the request.",
		}
	}
}
