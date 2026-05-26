module sov.fleet/s-natives

go 1.24

require (
	github.com/quic-go/quic-go v0.59.1
	sov.fleet/s-sacp v0.0.0
)

replace sov.fleet/s-sacp => ../s-sacp

require (
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
)
