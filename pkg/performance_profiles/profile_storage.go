package performance_profiles

import (
	"gopkg.in/mgo.v2"
	"log"
	"github.com/Cloud-Pie/SPDT/internal/types"
	"gopkg.in/mgo.v2/bson"
)

type PerformanceProfileDAO struct {
	Server	string
	Database	string
}

var db *mgo.Database
const COLLECTION = "performanceProfile"


//Connect to the database
func (p *PerformanceProfileDAO) Connect() {
	session, err := mgo.Dial(p.Server)
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB(p.Database)
}

//Retrieve all the stored elements
func (p *PerformanceProfileDAO) FindAll() ([]types.PerformanceProfile, error) {
	var performanceProfiles []types.PerformanceProfile
	err := db.C(COLLECTION).Find(bson.M{}).All(&performanceProfiles)
	return performanceProfiles, err
}

//Retrieve the item with the specified ID
func (p *PerformanceProfileDAO) FindByID(id string) (types.PerformanceProfile, error) {
	var performanceProfile types.PerformanceProfile
	err := db.C(COLLECTION).FindId(bson.ObjectIdHex(id)).One(&performanceProfile)
	return performanceProfile,err
}

//Insert a new Performance Profile
func (p *PerformanceProfileDAO) Insert(performanceProfile types.PerformanceProfile) error {
	err := db.C(COLLECTION).Insert(&performanceProfile)
	return err
}

//Delete the specified item
func (p *PerformanceProfileDAO) Delete(performanceProfile types.PerformanceProfile) error{
	err := db.C(COLLECTION).Remove(&performanceProfile)
	return err
}