package test

type Param struct {
}

type Result struct {
}

// +rest:gin
// +path=/test
type TestService interface {
	// +path=/paramAndRes
	// +method=post
	// +successCode=201
	GetParmAndRes(*Param) (*Result, error)
	// +path=/param
	// +method=get
	// +successCode=201
	GetParm(Param) error
	// +path=/res
	// +method=post
	// +successCode=201
	PostRes() (Result error)
}