package storage

import (
	"gopkg.in/mgo.v2"
	"github.com/Cloud-Pie/SPDT/types"
	"gopkg.in/mgo.v2/bson"
	"os"
	"time"
)

type VMBootingProfileDAO struct {
	Server     string
	Database   string
	Collection string
	db         *mgo.Database
	Session    *mgo.Session

}

var VMBootingProfileDB *VMBootingProfileDAO
var DEFAULT_DB_SERVER_VM_BOOTING = os.Getenv("PROFILESDB_HOST")
var profilesVMDBHost = []string{ DEFAULT_DB_SERVER_VM_BOOTING,}

const (
    DEFAULT_DB_COLLECTION_VM_PROFILES = "VM_Booting_Profile"
    )

//Connect to the database
func (p *VMBootingProfileDAO) Connect() (*mgo.Database, error) {
	var err error

	if p.Session == nil {
		p.Session,  err = mgo.DialWithInfo(&mgo.DialInfo{
			Addrs: profilesDBHost,
			Username: os.Getenv("PROFILESDB_USER"),
			Password: os.Getenv("PROFILESDB_PASS"),
			Timeout:  60 * time.Second,
		})
		if err != nil {
			return nil, err
		}
	}
	p.Session = p.Session.Clone()
	p.db = p.Session.DB(p.Database)
	return p.db,err
}

//Retrieve all the stored elements
func (p *VMBootingProfileDAO) FindAll() ([]types.InstancesBootShutdownTime, error) {
	var vmBootingProfiles []types.InstancesBootShutdownTime
	err := p.db.C(p.Collection).Find(bson.M{}).All(&vmBootingProfiles)
	return vmBootingProfiles, err
}

//Retrieve the item with the specified ID
func (p *VMBootingProfileDAO) FindByType(vmType string) (types.InstancesBootShutdownTime, error) {
	var vmBootingProfile types.InstancesBootShutdownTime
	err := p.db.C(p.Collection).Find(bson.M{"vm_type":vmType}).One(&vmBootingProfile)
	return vmBootingProfile,err
}

//Insert a new Performance Profile
func (p *VMBootingProfileDAO) Insert(vmBootingProfile types.InstancesBootShutdownTime) error {
	err := p.db.C(p.Collection).Insert(&vmBootingProfile)
	return err
}

//Update by type
func (p *VMBootingProfileDAO) UpdateByType(vmType string, vmBootingProfile types.InstancesBootShutdownTime) error {
	err := p.db.C(p.Collection).
		Update(bson.M{"vm_type":vmType},vmBootingProfile)
	return err
}


//Search booting and shutdown time for a vm type and number of instances
func (p *VMBootingProfileDAO) BootingShutdownTime(vmType string, numInstances int) (types.BootShutDownTime, error){
	type Result struct {
		BootShutDown types.BootShutDownTime `bson:"instances_values"`
	}
	var result Result
	query := []bson.M {
		bson.M{"$match" : bson.M{"vm_type" : vmType}},
		bson.M{"$unwind": "$instances_values" },
		bson.M{"$match": bson.M{"instances_values.num_instances": numInstances}}}
	err := p.db.C(p.Collection).Pipe(query).One(&result)
	return result.BootShutDown, err
}

//Search booting and shutdown time for a vm type
func (p *VMBootingProfileDAO) InstanceVMBootingShutdown(vmType string) (types.InstancesBootShutdownTime, error){
	var result types.InstancesBootShutdownTime
	query := []bson.M {
		bson.M{"$match" : bson.M{"vm_type" : vmType}}}
	err := p.db.C(p.Collection).Pipe(query).One(&result)
	return result, err
}

func GetVMBootingProfileDAO(serviceName string) *VMBootingProfileDAO {
	if VMBootingProfileDB == nil {
		VMBootingProfileDB = &VMBootingProfileDAO {
			Database:DEFAULT_DB_PROFILES,
			Collection:DEFAULT_DB_COLLECTION_VM_PROFILES + "_" + serviceName,
		}
		_,err := VMBootingProfileDB.Connect()
		if err != nil {
			log.Error("Error connecting to Profiles database "+err.Error())
		}
	}
	return VMBootingProfileDB
}