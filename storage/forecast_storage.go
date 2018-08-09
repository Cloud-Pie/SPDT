package storage

import (
	"gopkg.in/mgo.v2"
	"github.com/Cloud-Pie/SPDT/types"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/util"
	"log"
)

type ForecastDAO struct {
	Server	string
	Database	string
	db *mgo.Database
	session *mgo.Session
}
var ForecastDB *ForecastDAO

//Connect to the database
func (p *ForecastDAO) Connect() (*mgo.Database, error) {
	var err error
	if p.session == nil {
		p.session, err = mgo.Dial(p.Server)
		if err != nil {
			return nil, err
		}
	}
	p.db = p.session.DB(p.Database)
	return p.db,err
}

//Retrieve all the stored elements
func (p *ForecastDAO) FindAll() ([]types.Forecast, error) {
	var forecast []types.Forecast
	err := p.db.C(util.DEFAULT_DB_COLLECTION_FORECAST).Find(bson.M{}).All(&forecast)
	return forecast, err
}

//Retrieve the item with the specified ID
func (p *ForecastDAO) FindByID(id string) (types.Forecast, error) {
	var forecast types.Forecast
	err := p.db.C(util.DEFAULT_DB_COLLECTION_FORECAST).FindId(bson.ObjectIdHex(id)).One(&forecast)
	return forecast,err
}

//Insert a new forecast
func (p *ForecastDAO) Insert(forecast types.Forecast) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_FORECAST).Insert(&forecast)
	return err
}

//Delete the specified item
func (p *ForecastDAO) Delete(forecast types.Forecast) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_FORECAST).Remove(&forecast)
	return err
}

//Delete the specified item
func (p *ForecastDAO) Update(id bson.ObjectId, forecast types.Forecast) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_FORECAST).Update(bson.M{"_id":id}, bson.M{"forecasted_values":forecast.ForecastedValues,
													"window_time_start":forecast.TimeWindowStart,
													"window_time_end":forecast.TimeWindowEnd})
	return err
}

/*func (p *ForecastDAO) FindByDate(timestamp time.Time) (types.Forecast, error) {
	var forecast types.Forecast
	err := db.C(COLLECTION).FindId(bson.ObjectIdHex(id)).One(&forecast)
	return forecast,err
}*/

func GetForecastDAO() *ForecastDAO{
	if ForecastDB == nil {
		ForecastDB = &ForecastDAO {
			Server:util.DEFAULT_DB_SERVER_FORECAST,
			Database:util.DEFAULT_DB_FORECAST,
		}
		_,err := ForecastDB.Connect()
		if err != nil {
			log.Fatalf(err.Error())
		}
	}
	return ForecastDB
}
