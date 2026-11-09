module neuroplast.io/myelin

go 1.24.0

require (
	github.com/tinylib/msgp v1.6.5
	neuroplast.io/cel-lite v0.2.0
	neuroplast.io/myelin/myelinkey v0.2.0
)

require github.com/philhofer/fwd v1.2.0 // indirect

replace neuroplast.io/myelin/myelinkey => ./myelinkey
