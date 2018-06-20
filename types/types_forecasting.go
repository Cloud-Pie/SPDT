package types

import (
	"time"
)

/*Critical Interval is the interval of time analyzed to take a scaling decision*/
type CriticalInterval struct {
	TimeStart	time.Time	`json:"TimeStart"`
	Requests	int	`json:"Requests"`	//max/min point in the interval
	AboveThreshold	bool `json:"AboveThreshold"`	//1:= aboveThreshold; -1:= below
	TimeEnd	time.Time	`json:"TimeEnd"`
	TimePeak time.Time
}

/*Represent the number of requests for a time T*/
type ForecastedValue struct {
	TimeStamp   time.Time	`json:"time-stamp"`
	Requests	int         `json:"requests"`
}

/*Set of values received from from the Forecasting component*/
type Forecast struct {
	ID	string							`json:"id"`
	ForecastedValues []ForecastedValue	`json:"values"`
}

/*ProcessedForecast metadata after processing the time serie*/
type ProcessedForecast struct {
	NeedToScale       bool
	CriticalIntervals [] CriticalInterval
	RawForecast		Forecast
}

/*Points of Interest*/
type PoI struct {
	Peak			bool	 `json:"peak"`
	Index 			int  	 `json:"index"`
	Left_ips		float64	 `json:"left_ips"`
	Right_ips		float64	 `json:"right_ips"`
	Widht_heights	float64	 `json:"widht_heights"`
	Index_in_interval []int	 `json:"index_in_interval"`
	Start  struct {
		Index		int			`json:"index"`
		Left_ips	float64		`json:"left_ips"`
		Right_ips	float64		`json:"right_ips"`
		Widht_heights	float64	`json:"widht_heights"`
	}							`json:"index_left_valley"`
	End  struct {
		Index		int			`json:"index"`
		Left_ips	float64		`json:"left_ips"`
		Right_ips	float64		`json:"right_ips"`
		Widht_heights	float64	`json:"widht_heights"`
	}							`json:"index_right_valley"`

}
