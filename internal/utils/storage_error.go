package utils

import "strings"

// SanitizeStorageConnectivityError converts a raw storage connectivity error
// into a safe, user-facing message. It deliberately avoids echoing the raw
// driver/network error so responses never leak internal hostnames, IPs, ports
// or TLS/certificate details. Callers that need the full error must log it
// server-side instead of returning it to the client.
func SanitizeStorageConnectivityError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "Endpoint url cannot have fully qualified paths"):
		return "Invalid Endpoint format: remove the http:// or https:// prefix and enter only the host or IP and port (e.g. minio.example.com:9000)"
	case strings.Contains(msg, "no such host"):
		return "DNS resolution failed; please check that the address is correct"
	case strings.Contains(msg, "connection refused"):
		return "Connection refused; please confirm the service is running and the port is correct"
	case strings.Contains(msg, "no route to host"):
		return "Unable to route to the target address; please check network configuration"
	case strings.Contains(msg, "i/o timeout") || strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "context deadline"):
		return "Connection timed out; please check the network or service status"
	case strings.Contains(msg, "403") || strings.Contains(msg, "AccessDenied") || strings.Contains(msg, "access denied"):
		return "Authentication failed; please check that the access credentials are correct"
	case strings.Contains(msg, "certificate") || strings.Contains(msg, "tls") || strings.Contains(msg, "x509"):
		return "TLS/SSL certificate error; please check the SSL configuration"
	case strings.Contains(msg, "404") || strings.Contains(msg, "NoSuchBucket"):
		return "Bucket does not exist; please check the name and Region"
	default:
		return "Connection failed; please check that the configuration is correct"
	}
}
