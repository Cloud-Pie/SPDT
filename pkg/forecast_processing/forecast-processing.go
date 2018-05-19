package forecast_processing

import (
	"github.com/yemramirezca/SPDT/internal/types"
	"fmt"
)

func ProcessData() types.Forecast {

	fmt.Println("start process")
	//TODO: Process the Time Serie
	criticalIntervals := [] types.CriticalInterval {}
	need_to_scale := ( len(criticalIntervals) == 0)
	return types.Forecast{need_to_scale, criticalIntervals}
}