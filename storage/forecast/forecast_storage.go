package policies_derivation

import (
	"gopkg.in/mgo.v2"
	"log"
	"github.com/Cloud-Pie/SPDT/types"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/util"
)

type ForecastDAO struct {
	Server	string
	Database	string
}

var db *mgo.Database
const COLLECTION = "Forecast"


//Connect to the database
func (p *ForecastDAO) Connect() {
	session, err := mgo.Dial(p.Server)
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB(p.Database)
}

//Retrieve all the stored elements
func (p *ForecastDAO) FindAll() ([]types.Forecast, error) {
	var forecast []types.Forecast
	err := db.C(COLLECTION).Find(bson.M{}).All(&forecast)
	return forecast, err
}

//Retrieve the item with the specified ID
func (p *ForecastDAO) FindByID(id string) (types.Forecast, error) {
	var forecast types.Forecast
	err := db.C(COLLECTION).FindId(bson.ObjectIdHex(id)).One(&forecast)
	return forecast,err
}

//Insert a new forecast
func (p *ForecastDAO) Insert(forecast types.Forecast) error {
	err := db.C(COLLECTION).Insert(&forecast)
	return err
}

//Delete the specified item
func (p *ForecastDAO) Delete(forecast types.Forecast) error {
	err := db.C(COLLECTION).Remove(&forecast)
	return err
}

func Store(forecast types.Forecast){
	//Store received information about forecasts
	forecastDAO := ForecastDAO{
		util.DEFAULT_DB_SERVER_FORECAST,
		util.DEFAULT_DB_FORECAST,
	}
	forecastDAO.Connect()

	err := forecastDAO.Insert(forecast)
	if err != nil {
		log.Fatalf(err.Error())
	}
}