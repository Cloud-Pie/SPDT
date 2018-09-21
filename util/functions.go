package util

import "strings"

/* Parses the parameters given in the endpoint template and replaces them for real values
	in:
		@endpoint string to parse
		@parameters map with key-values
	out:
		@string	Endpoint with new values
*/
func ParseURL(endpoint string, parameters map[string]string) string {
	for key, value := range parameters {
		endpoint = strings.Replace(endpoint, "{"+key+"}", value, -1)
	}
	return endpoint
}
