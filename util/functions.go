package util

import (
	"strings"
	"math"
)

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

/* Utility function to convert from milliseconds to seconds
	in:
		@m  miliseconds
	out:
		@float	seconds
*/
func MillisecondsToSeconds(m float64) float64{
	return m/1000
}

/* Round a float to n decimals
	in:
		@value  value to be rounded
		@decimals  number of decimals after coma
	out:
		@float	new value rounded
*/
func RoundN(value float64, decimals float64) float64 {
	factor := math.Pow(10, decimals)
	roundedValue := math.Ceil(value*factor)/factor
	return roundedValue
}