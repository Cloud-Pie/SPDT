package storage

import (
	"gopkg.in/mgo.v2"
	"github.com/Cloud-Pie/SPDT/types"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/util"
)

var PerformanceProfileDB *PerformanceProfileDAO

type PerformanceProfileDAO struct {
	Server		string
	Database	string
	db 			*mgo.Database
	session *mgo.Session

}

//Connect to the database
func (p *PerformanceProfileDAO) Connect() (*mgo.Database, error) {
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
func (p *PerformanceProfileDAO) FindAll() ([]types.ServiceProfile, error) {
	var performanceProfiles []types.ServiceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{}).All(&performanceProfiles)
	return performanceProfiles, err
}

//Retrieve the item with the specified ID
func (p *PerformanceProfileDAO) FindByID(id string) (types.ServiceProfile, error) {
	var performanceProfile types.ServiceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).FindId(bson.ObjectIdHex(id)).One(&performanceProfile)
	return performanceProfile,err
}

//Retrieve the item with the specified ID
func (p *PerformanceProfileDAO) FindByAppName(name string) (types.ServiceProfile, error) {
	var performanceProfile types.ServiceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{"name": name}).One(&performanceProfile)
	return performanceProfile,err
}

//Insert a new Performance Profile
func (p *PerformanceProfileDAO) Insert(performanceProfile types.ServiceProfile) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Insert(&performanceProfile)
	return err
}

//Delete the specified item
func (p *PerformanceProfileDAO) Delete(performanceProfile types.ServiceProfile) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Remove(&performanceProfile)
	return err
}


func (p *PerformanceProfileDAO) FindByLimitsOver(cores float64, memory float64, requests float64) (types.PerformanceProfile, error) {
	//db.getCollection('trnProfiles').find({"limits.cpu_cores" : 1000,"limits.mem_gb" : 500, "trns": {$elemMatch:{"replicas":2} } }, {_id: 0, "trns.$":1})
	var performanceProfile types.PerformanceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{
					"limits.cpu_cores" : cores,
					 "limits.mem_gb" : memory,
					"trns": bson.M{"$elemMatch": bson.M{"maximum_service_capacity_per_sec":bson.M{"$gte": requests}}}}).
		Select(bson.M{"_id": 0, "limits":1, "trns.$":1}).One(&performanceProfile)
	return performanceProfile,err
}


func (p *PerformanceProfileDAO) FindByLimitsUnder(cores float64, memory float64, requests float64) (types.PerformanceProfile, error) {
	var performanceProfile types.PerformanceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{
		"limits.cpu_cores" : cores,
		"limits.mem_gb" : memory,
		"trns": bson.M{"$elemMatch": bson.M{"maximum_service_capacity_per_sec":bson.M{"$lt": requests}}}}).
		Select(bson.M{"_id": 0, "limits":1, "trns.$":1}).One(&performanceProfile)
	return performanceProfile,err
}

func (p *PerformanceProfileDAO) FindNewLimitsOver(requests float64) ([]types.PerformanceProfile, error) {
	var performanceProfile []types.PerformanceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{
		"trns": bson.M{"$elemMatch": bson.M{"maximum_service_capacity_per_sec":bson.M{"$gte": requests}}}}).
		Select(bson.M{"_id": 0, "limits":1, "trns.$":1}).All(&performanceProfile)
	return performanceProfile,err
}

func (p *PerformanceProfileDAO) FindNewLimitsUnder(requests float64) ([]types.PerformanceProfile, error) {
	var performanceProfile []types.PerformanceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{
		"trns": bson.M{"$elemMatch": bson.M{"maximum_service_capacity_per_sec":bson.M{"$lt": requests}}}}).
		Select(bson.M{"_id": 0, "limits":1, "trns.$":1}).All(&performanceProfile)
	return performanceProfile,err
}

func (p *PerformanceProfileDAO) FindProfileTRN(cores float64, memory float64, numberReplicas int) (types.PerformanceProfile, error) {
	var performanceProfile types.PerformanceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{
		"limits.cpu_cores" : cores,
		"limits.mem_gb" : memory,
		"trns": bson.M{"$elemMatch": bson.M{"replicas":bson.M{"$gte": numberReplicas}}}}).
		Select(bson.M{"_id": 0, "limits":1, "trns.$":1}).One(&performanceProfile)
	return performanceProfile,err
}

func GetPerformanceProfileDAO() *PerformanceProfileDAO {
	if PerformanceProfileDB == nil {
		PerformanceProfileDB = &PerformanceProfileDAO {
			Server:util.DEFAULT_DB_SERVER_PROFILES,
			Database:util.DEFAULT_DB_PROFILES,
		}
		_,err := PerformanceProfileDB.Connect()
		if err != nil {
			log.Error(err.Error())
		}
	}
	return PerformanceProfileDB
}