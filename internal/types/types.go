package types

//Service keeps the name and scale of the scaled service.
type Service struct {
	Name  string
	Scale int
}

//State is the metadata of the state expected to scale to.
type State struct {
	Time     string
	Services []Service
	Name     string
}

type VM struct {
	Vm_type string `json:"vm_type"`
	Trn int `json:"trn"`
	Num_cores int `json:"num_cores"`
	Memory_gb int `json:"memory_gb"`
}

type PriceVM struct {
	Price	float32
	TimeUnit	float32
	VM	VM
}

type Policy struct {
	Total_cost float32
	States 	[] State
}

type CriticalInterval struct {
	TimeStart	string
	Requests	int	//max/min point in the interval
	Trend	int //1:= aboveThreshold; -1:= below
	TimeEnd	string
}

type Forecast struct {
	Need_to_scale bool
	CriticalIntervals [] CriticalInterval
}