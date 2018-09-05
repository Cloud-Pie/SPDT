package storage

import (
	"gopkg.in/mgo.v2"
	"github.com/Cloud-Pie/SPDT/types"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/util"
	"errors"
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
func (p *PerformanceProfileDAO) FindAll() ([]types.PerformanceProfile, error) {
	var performanceProfiles []types.PerformanceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{}).All(&performanceProfiles)
	return performanceProfiles, err
}

//Retrieve the item with the specified ID
func (p *PerformanceProfileDAO) FindByID(id string) (types.PerformanceProfile, error) {
	var performanceProfile types.PerformanceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).FindId(bson.ObjectIdHex(id)).One(&performanceProfile)
	return performanceProfile,err
}

//Insert a new Performance Profile
func (p *PerformanceProfileDAO) Insert(performanceProfile types.PerformanceProfile) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Insert(&performanceProfile)
	return err
}

//Delete the specified item
func (p *PerformanceProfileDAO) Delete(performanceProfile types.PerformanceProfile) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Remove(&performanceProfile)
	return err
}

//Update by id
func (p *PerformanceProfileDAO) UpdateById(id bson.ObjectId, performanceProfile types.PerformanceProfile) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).
		Update(bson.M{"_id":id},performanceProfile)
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

/*
	Matches the profiles that have exactly the resource limits specified as input and that provide a TRN greater or equal
	than the number of requests needed
	in:
		@cores float64
		@memory float64
		@requests float64
	out:
		@ContainersConfig
		@error
*/
func (p *PerformanceProfileDAO) MatchByLimitsOver(cores float64, memory float64, requests float64) (types.ContainersConfig, error){
	var result types.ContainersConfig
	query := []bson.M{
		bson.M{"$match" : bson.M{"limits.cpu_cores" : cores,"limits.mem_gb" : memory} },
		bson.M{"$unwind": "$trns" },
		bson.M{"$match": bson.M{"trns.maximum_service_capacity_per_sec":bson.M{"$gte": requests}}},
		bson.M{"$sort": bson.M{"trns.maximum_service_capacity_per_sec": 1}}}
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Pipe(query).One(&result)
	return result, err
}

/*
	Matches the profiles that have exactly the resource limits specified as input and that provide a TRN less
	than the number of requests needed
	in:
		@cores float64
		@memory float64
		@requests float64
	out:
		@ContainersConfig
		@error
*/
func (p *PerformanceProfileDAO) MatchByLimitsUnder(cores float64, memory float64, requests float64) (types.ContainersConfig, error){
	var result types.ContainersConfig
	query := []bson.M{
		bson.M{"$match" : bson.M{"limits.cpu_cores" : cores,"limits.mem_gb" : memory} },
		bson.M{"$unwind": "$trns" },
		bson.M{"$match": bson.M{"trns.maximum_service_capacity_per_sec":bson.M{"$lt": requests}}},
		bson.M{"$sort": bson.M{"trns.maximum_service_capacity_per_sec": -1}}}
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Pipe(query).One(&result)
	return result, err
}

/*
	Matches the profiles without consider restriction of limits that provide a TRN greater or equal
	than the number of requests needed
	in:
		@requests float64
	out:
		@ContainersConfig []types.ContainersConfig
		@error

*/
func (p *PerformanceProfileDAO) MatchOver(requests float64) ([]types.ContainersConfig, error) {
	var result []types.ContainersConfig
	query := []bson.M{
		bson.M{"$unwind": "$trns" },
		bson.M{"$match": bson.M{"trns.maximum_service_capacity_per_sec":bson.M{"$gte": requests}}},
		bson.M{"$sort": bson.M{"limits.cpu_cores":1, "limits.mem_gb":1, "trns.replicas":1, "trns.maximum_service_capacity_per_sec": 1}}}
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Pipe(query).All(&result)
	return result, err
}

/*
	Matches the profiles without consider restriction of limits that provide a TRN less than
	than the number of requests needed
	in:
		@requests float64
	out:
		@ContainersConfig []types.ContainersConfig
		@error
*/
func (p *PerformanceProfileDAO) MatchUnder(requests float64) ([]types.ContainersConfig, error) {
	var result []types.ContainersConfig
	query := []bson.M{
		bson.M{"$unwind": "$trns" },
		bson.M{"$match": bson.M{"trns.maximum_service_capacity_per_sec":bson.M{"$lt": requests}}},
		bson.M{"$sort": bson.M{"limits.cpu_cores":1, "limits.mem_gb":1, "trns.replicas":1, "trns.maximum_service_capacity_per_sec":-1}}}
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Pipe(query).All(&result)
	return result, err
}

/*
	Matches the profiles  which fit into the specified limits and that provide a TRN greater or equal than
	than the number of requests needed
	in:
		@requests float64
	out:
		@ContainersConfig []types.ContainersConfig
		@error
*/
func (p *PerformanceProfileDAO) MatchProfileFitLimitsOver(cores float64, memory float64, requests float64) ([]types.ContainersConfig, error) {
	var result []types.ContainersConfig
	query := []bson.M{
		bson.M{ "$match" : bson.M{"limits.cpu_cores" : bson.M{"$lte": cores}, "limits.mem_gb" : bson.M{"$lte":memory}}},
		bson.M{"$unwind": "$trns" },
		bson.M{"$match": bson.M{"trns.maximum_service_capacity_per_sec":bson.M{"$gte": requests}}},
		bson.M{"$sort": bson.M{"limits.cpu_cores":1, "limits.mem_gb":1, "trns.replicas":1, "trns.maximum_service_capacity_per_sec": 1}}}
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Pipe(query).All(&result)
	if len(result) == 0 {
		return result, errors.New("No result found")
	}
	return result, err
}

/*
	Matches the profiles  which fit into the specified limits and that provide a TRN less than
	than the number of requests needed
	in:
		@requests float64
	out:
		@ContainersConfig []types.ContainersConfig
		@error
*/
func (p *PerformanceProfileDAO) MatchProfileFitLimitsUnder(cores float64, memory float64,requests float64) ([]types.ContainersConfig, error) {
	var result []types.ContainersConfig
	query := []bson.M{
		bson.M{ "$match" : bson.M{"limits.cpu_cores" : bson.M{"$lte": cores}, "limits.mem_gb" : bson.M{"$lte":memory}}},
		bson.M{"$unwind": "$trns" },
		bson.M{"$match": bson.M{"trns.maximum_service_capacity_per_sec":bson.M{"$lt": requests}}},
		bson.M{"$sort": bson.M{"limits.cpu_cores":1, "limits.mem_gb":1, "trns.replicas":1, "trns.maximum_service_capacity_per_sec":-1}}}
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Pipe(query).All(&result)
	if len(result) == 0 {
		return result, errors.New("No result found")
	}
	return result, err
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

func (p *PerformanceProfileDAO) FindProfileByLimits(limit types.Limit) (types.PerformanceProfile, error) {
	var performanceProfile types.PerformanceProfile
	err := p.db.C(util.DEFAULT_DB_COLLECTION_PROFILES).Find(bson.M{
		"limits.cpu_cores" : limit.NumberCores,
		"limits.mem_gb" : limit.MemoryGB}).One(&performanceProfile)
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