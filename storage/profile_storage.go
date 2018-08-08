package storage

import (
	"gopkg.in/mgo.v2"
	"github.com/Cloud-Pie/SPDT/types"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/util"
)

type PerformanceProfileDAO struct {
	Server		string
	Database	string
	db 			*mgo.Database
}


//Connect to the database
func (p *PerformanceProfileDAO) Connect() (*mgo.Database, error) {
	session, err := mgo.Dial(p.Server)
	if err != nil {
		return nil, err
	}
	p.db = session.DB(p.Database)
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