package types

import (
	"time"
	"gopkg.in/mgo.v2/bson"
)

/*Critical Interval is the interval of time analyzed to take a scaling decision*/
type CriticalInterval struct {
	TimeStart	time.Time	`json:"TimeStart"`
	Requests	float64	`json:"Requests"`	//max/min point in the interval
	AboveThreshold	bool `json:"AboveThreshold"`	//1:= aboveThreshold; -1:= below
	TimeEnd	time.Time	`json:"TimeEnd"`
	TimePeak time.Time
}

/*Represent the number of requests for a time T*/
type ForecastedValue struct {
	TimeStamp   time.Time	`json:"timestamp"`
	Requests	float64         `json:"requests"`
}

/*Set of values received from the Forecasting component*/
type Forecast struct {
	IDdb             bson.ObjectId     `bson:"_id"`
	ServiceName		 string			   `json:"service_name"  bson:"service_name"`
	ForecastedValues []ForecastedValue `json:"values"  bson:"values"`
	TimeWindowStart  time.Time         `json:"start_time"  bson:"start_time"`
	TimeWindowEnd    time.Time         `json:"end_time"  bson:"end_time"`
	IDPrediction     string            `json:"id"  bson:"id_predictions"`
}

/*ProcessedForecast metadata after processing the time serie*/
type ProcessedForecast struct {
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
	Index_in_interval_right []int	 `json:"index_in_interval_right"`
	Index_in_interval_left []int	 `json:"index_in_interval_left"`
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
