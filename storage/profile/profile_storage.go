package profile

import (
	"gopkg.in/mgo.v2"
	"log"
	"github.com/Cloud-Pie/SPDT/types"
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
func (p *PerformanceProfileDAO) FindAll() ([]types.ServiceProfile, error) {
	var performanceProfiles []types.ServiceProfile
	err := db.C(COLLECTION).Find(bson.M{}).All(&performanceProfiles)
	return performanceProfiles, err
}

//Retrieve the item with the specified ID
func (p *PerformanceProfileDAO) FindByID(id string) (types.ServiceProfile, error) {
	var performanceProfile types.ServiceProfile
	err := db.C(COLLECTION).FindId(bson.ObjectIdHex(id)).One(&performanceProfile)
	return performanceProfile,err
}

//Insert a new Performance Profile
func (p *PerformanceProfileDAO) Insert(performanceProfile types.ServiceProfile) error {
	err := db.C(COLLECTION).Insert(&performanceProfile)
	return err
}

//Delete the specified item
func (p *PerformanceProfileDAO) Delete(performanceProfile types.ServiceProfile) error{
	err := db.C(COLLECTION).Remove(&performanceProfile)
	return err
}