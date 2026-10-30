module github.com/neuroplastio/myelin

go 1.24.0

require (
	github.com/neuroplastio/cel-lite v0.1.0
	github.com/neuroplastio/myelin/myelinkey v0.1.0-next.1
	github.com/tinylib/msgp v1.6.5
)

require github.com/philhofer/fwd v1.2.0 // indirect

replace github.com/neuroplastio/myelin/myelinkey => ./myelinkey
