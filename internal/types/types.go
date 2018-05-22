package types

//Service keeps the name and scale of the scaled service.
type Service struct {
	Name  string
	Scale int
}

//State is the metadata of the state expected to scale to.
type State struct {
	Time     string
	Services [] Service
	Name     string
	VmsScale  [] VmScale
}

type VmScale struct {
	Type string `json:"type"`
	Scale int `json:"scale"`
}



type Policy struct {
	ID string
	TotalCost float32
	States    [] State
}

type CriticalInterval struct {
	TimeStart	string	`json:"TimeStart"`
	Requests	int	`json:"Requests"`	//max/min point in the interval
	Trend	int `json:"Trend"`	//1:= aboveThreshold; -1:= below
	TimeEnd	string	`json:"TimeEnd"`
}

type Forecast struct {
	NeedToScale       bool
	CriticalIntervals [] CriticalInterval
}