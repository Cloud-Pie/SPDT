package forecast_processing

import (
	"fmt"
	"github.com/Cloud-Pie/SPDT/internal/types"
)

func ProcessData() types.Forecast {
	fmt.Println("start process")
	//TODO: Process the Time Serie

	return getMockData()
}

