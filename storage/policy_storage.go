package storage

import (
	"gopkg.in/mgo.v2"
	"github.com/Cloud-Pie/SPDT/types"
	"gopkg.in/mgo.v2/bson"
	"time"
	"github.com/Cloud-Pie/SPDT/util"
)

var PolicyDB *PolicyDAO

type PolicyDAO struct {
	Server	string
	Database	string
	db *mgo.Database
	session *mgo.Session
}

//Connect to the database
func (p *PolicyDAO) Connect() (*mgo.Database, error) {
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
func (p *PolicyDAO) FindAll() ([]types.Policy, error) {
	var policies []types.Policy
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).Find(bson.M{}).All(&policies)
	return policies, err
}

//Retrieve the item with the specified ID
func (p *PolicyDAO) FindByID(id string) (types.Policy, error) {
	var policies types.Policy
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).FindId(bson.ObjectIdHex(id)).One(&policies)
	return policies,err
}

//Retrieve all policies for start time greater than or equal to time t
func (p *PolicyDAO) FindByStartTime(time time.Time) ([]types.Policy, error) {
	var policies []types.Policy
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).
		Find(bson.M{"window_time_start": bson.M{"$gte":time}}).All(&policies)
	return policies,err
}

//Retrieve all policies for start time less than or equal to time t
func (p *PolicyDAO) FindByEndTime(time time.Time) ([]types.Policy, error) {
	var policies []types.Policy
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).
		Find(bson.M{"window_time_end": bson.M{"$lte":time}}).All(&policies)
	return policies,err
}

//Retrieve all policies for start time greater than or equal to time t
func (p *PolicyDAO) FindAllByTimeWindow(startTime time.Time, endTime time.Time) ([]types.Policy, error) {
	var policies []types.Policy
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).
		Find(bson.M{"window_time_start": bson.M{"$gte":startTime},
					"window_time_end": bson.M{"$lte":endTime}}).All(&policies)
	return policies,err
}

//Retrieve all policies for start time greater than or equal to time t
func (p *PolicyDAO) FindOneByTimeWindow(startTime time.Time, endTime time.Time) (types.Policy, error) {
	var policy types.Policy
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).
		Find(bson.M{"window_time_start": bson.M{"$eq":startTime},
		            "window_time_end": bson.M{"$eq":endTime}}).One(&policy)
	return policy,err
}

//Insert a new Performance Profile
func (p *PolicyDAO) Insert(policies types.Policy) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).Insert(&policies)
	return err
}

//Delete policy by id
func (p *PolicyDAO) DeleteById(id string) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).RemoveId(bson.ObjectIdHex(id))
	return err
}

//Delete policy for the time window
func (p *PolicyDAO) DeleteOneByTimeWindow(startTime time.Time, endTime time.Time) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).
		Remove(bson.M{"window_time_start": bson.M{"$eq":startTime},
		              "window_time_end": bson.M{"$eq":startTime}})
	return err
}

//Update policy by id
func (p *PolicyDAO) UpdateById(id bson.ObjectId, policy types.Policy) error {
	err := p.db.C(util.DEFAULT_DB_COLLECTION_POLICIES).
		Update(bson.M{"_id":id},policy)
	return err
}

func GetPolicyDAO() *PolicyDAO{
	if PolicyDB == nil {
		PolicyDB = &PolicyDAO {
			Server:util.DEFAULT_DB_SERVER_POLICIES,
			Database:util.DEFAULT_DB_POLICIES,
		}
		_,err := PolicyDB.Connect()
		if err != nil {
			log.Error(err.Error())
		}
	}
	return PolicyDB
}