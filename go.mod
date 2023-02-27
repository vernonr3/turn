module github.com/pion/turn/v2

go 1.13

replace github.com/pion/turn/v2/internal/ipnet => ./internal/ipnet

replace github.com/pion/stun => ../stun

require (
	github.com/kr/pretty v0.1.0 // indirect
	github.com/pion/logging v0.2.2
	github.com/pion/randutil v0.1.0
	github.com/pion/stun v0.4.0
	github.com/pion/transport/v2 v2.0.1
	github.com/stretchr/testify v1.8.1
	golang.org/x/sys v0.5.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)
