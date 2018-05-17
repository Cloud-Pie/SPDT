package types

//Service keeps the name and scale of the scaled service.
type Service struct {
	Name  string
	Scale string
}

//State is the metadata of the state expected to scale to.
type State struct {
	Time     string
	Services []Service
	Name     string
}
//trend, policy, vm, resource,
type VMProfile struct {
	vm_type string
	trn int
}

type Policy struct {

}

type Forecast struct {
	Need_to_scale bool
}