package policies_derivation

import (
	"gopkg.in/mgo.v2"
	"log"
	"github.com/Cloud-Pie/SPDT/types"
	"gopkg.in/mgo.v2/bson"
	"github.com/Cloud-Pie/SPDT/util"
	"time"
)

type PolicyDAO struct {
	Server	string
	Database	string
}

var db *mgo.Database
const COLLECTION = "Policies"


//Connect to the database
func (p *PolicyDAO) Connect() {
	session, err := mgo.Dial(p.Server)
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB(p.Database)
}

//Retrieve all the stored elements
func (p *PolicyDAO) FindAll() ([]types.Policy, error) {
	var policies []types.Policy
	err := db.C(COLLECTION).Find(bson.M{}).All(&policies)
	return policies, err
}

//Retrieve the item with the specified ID
func (p *PolicyDAO) FindByID(id string) (types.Policy, error) {
	var policies types.Policy
	err := db.C(COLLECTION).FindId(bson.ObjectIdHex(id)).One(&policies)
	return policies,err
}

//Retrieve the item that starts at time t
func (p *PolicyDAO) FindByStartTime(time time.Time) (types.Policy, error) {
	var policies types.Policy
	err := db.C(COLLECTION).FindId(bson.M{"window_start_time": time}).One(&policies)
	return policies,err
}

//Insert a new Performance Profile
func (p *PolicyDAO) Insert(policies types.Policy) error {
	err := db.C(COLLECTION).Insert(&policies)
	return err
}

//Delete the specified item
func (p *PolicyDAO) Delete(policies types.Policy) error {
	err := db.C(COLLECTION).Remove(&policies)
	return err
}

func Store(policy types.Policy){
	//Store received information about Performance Profiles
	//policy.ID = bson.NewObjectId()
	policyDAO := PolicyDAO{
		util.DEFAULT_DB_SERVER_POLICIES,
		util.DEFAULT_DB_POLICIES,
	}
	policyDAO.Connect()

	err := policyDAO.Insert(policy)
	if err != nil {
		log.Fatalf(err.Error())
	}
}