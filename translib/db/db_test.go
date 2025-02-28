////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
//  its subsidiaries.                                                         //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//     http://www.apache.org/licenses/LICENSE-2.0                             //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

package db


import (
	// "fmt"
	// "errors"
	// "flag"
	// "github.com/golang/glog"
	"time"
	"io/ioutil"
	"os"
	"testing"
	"strconv"
	"reflect"
)

var dbConfig = `
{
    "INSTANCES": {
        "redis":{
            "hostname" : "127.0.0.1",
            "port" : 6379,
            "unix_socket_path" : "/var/run/redis/redis.sock",
            "persistence_for_warm_boot" : "yes"
        },
        "redis2":{
            "hostname" : "127.0.0.1",
            "port" : 63792,
            "unix_socket_path" : "/var/run/redis/redis2.sock",
            "persistence_for_warm_boot" : "yes"
        },
        "redis3":{
           "hostname" : "127.0.0.1",
            "port" : 63793,
            "unix_socket_path" : "/var/run/redis/redis3.sock",
            "persistence_for_warm_boot" : "yes"
        },
        "rediswb":{
            "hostname" : "127.0.0.1",
            "port" : 63970,
            "unix_socket_path" : "/var/run/redis/rediswb.sock",
            "persistence_for_warm_boot" : "yes"
        }
    },
    "DATABASES" : {
        "APPL_DB" : {
            "id" : 0,
            "separator": ":",
            "instance" : "redis2"
        },
        "ASIC_DB" : {
            "id" : 1,
            "separator": ":",
            "instance" : "redis3"
        },
        "COUNTERS_DB" : {
            "id" : 2,
            "separator": ":",
            "instance" : "redis"
        },
        "LOGLEVEL_DB" : {
            "id" : 3,
            "separator": ":",
            "instance" : "redis"
        },
        "CONFIG_DB" : {
            "id" : 4,
            "separator": "|",
            "instance" : "redis"
        },
        "PFC_WD_DB" : {
            "id" : 5,
            "separator": ":",
            "instance" : "redis"
        },
        "FLEX_COUNTER_DB" : {
            "id" : 5,
            "separator": ":",
            "instance" : "redis"
        },
        "STATE_DB" : {
            "id" : 6,
            "separator": "|",
            "instance" : "redis"
        },
        "SNMP_OVERLAY_DB" : {
            "id" : 7,
            "separator": "|",
            "instance" : "redis"
        },
        "ERROR_DB" : {
            "id" : 8,
            "separator": ":",
            "instance" : "redis"
        }
    },
    "VERSION" : "1.0"
}
`


func TestMain(m * testing.M) {

	exitCode := 0

/* Apparently, on an actual switch the swss container will have
 * a redis-server running, which will be in a different container than
 * mgmt, thus this pkill stuff to find out it is running will not work.
 *

	redisServerAttemptedStart := false

TestMainRedo:
	o, e := exec.Command("/usr/bin/pkill", "-HUP", "redis-server").Output()

	if e == nil {

	} else if redisServerAttemptedStart {

		exitCode = 1

	} else {

		fmt.Printf("TestMain: No redis server: pkill: %v\n", o)
		fmt.Println("TestMain: Starting redis-server")
		e = exec.Command("/tools/bin/redis-server").Start()
		time.Sleep(3 * time.Second)
		redisServerAttemptedStart = true
		goto TestMainRedo
	}
*/

	// Create Temporary DB Config File
	dbContent := []byte(dbConfig)
	dbFile, e := ioutil.TempFile("/tmp", "dbConfig")
	if e != nil {
		exitCode = 1
	} else {
		defer os.Remove(dbFile.Name())
	}

	if _,e := dbFile.Write(dbContent); e != nil {
		exitCode = 2
	}

	if e := dbFile.Close(); e != nil {
		exitCode = 3
	}

	// Set the environment variable to it
	os.Setenv("DB_CONFIG_PATH", dbFile.Name())

	if exitCode == 0 {
		exitCode = m.Run()
	}


	os.Exit(exitCode)
	
}

func initMultiAsic() {
	os.Setenv("ASIC_CONFIG_PATH", "data/asic_multi.conf")
	defer os.Unsetenv("ASIC_CONFIG_PATH")
	os.Setenv("DB_GLOBAL_CONFIG_PATH", "data/database_global.json")
	defer os.Unsetenv("DB_GLOBAL_CONFIG_PATH")
	initAllDbs()
}

func initNotMultiAsic() {
	os.Setenv("ASIC_CONFIG_PATH", "data/asic.conf")
	defer os.Unsetenv("ASIC_CONFIG_PATH")
	os.Setenv("DB_CONFIG_PATH", "data/database_config.json")
	defer os.Unsetenv("DB_CONFIG_PATH")
	initAllDbs()
}

func TestIsMultiAsicFalse(t *testing.T) {
	initNotMultiAsic()

	if isMultiAsic() {
		t.Errorf("expected is not multi asic but got ture")
	}
}

func TestIsMultiAsicTrue(t *testing.T) {
	initMultiAsic()

	if !isMultiAsic() {
		t.Errorf("expected is multi asic but got false")
	}
}

func TestHostDbWhenIsNotMultiAsic(t *testing.T) {
	initNotMultiAsic()

	mDbs, err := GetMDBInstances(true)
	if err != nil {
		t.Errorf("failed to get all db instances")
	}

	if len(mDbs) != 1 {
		t.Errorf("failed to get host db instance")
	}

	if mDbs["host"][ApplDB] == nil ||
		mDbs["host"][StateDB] == nil ||
		mDbs["host"][CountersDB] == nil {
		t.Errorf("the host db instance is empty")
	}

	if mDbs["host"][ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("host db instance's address is wrong")
	}
}

func TestGetAllDbInstancesWhenItsMultiAsic(t *testing.T) {
	initMultiAsic()

	mDbs, err := GetMDBInstances(true)
	if err != nil {
		t.Errorf("failed to get all db instances")
	}

	if len(mDbs) != 5 {
		t.Errorf("failed to get host db instance")
	}

	if mDbs["host"][ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("host db instance's address is wrong")
	}

	if mDbs["asic0"][ApplDB].client.Options().Addr != "127.0.0.10:6379" {
		t.Errorf("asic0 db instance's address is wrong")
	}

	if mDbs["asic1"][ApplDB].client.Options().Addr != "127.0.0.11:6379" {
		t.Errorf("asic1 db instance's address is wrong")
	}

	if mDbs["asic2"][ApplDB].client.Options().Addr != "127.0.0.12:6379" {
		t.Errorf("asic2 db instance's address is wrong")
	}

	if mDbs["asic3"][ApplDB].client.Options().Addr != "127.0.0.13:6379" {
		t.Errorf("asic3 db instance's address is wrong")
	}
}

func TestGetSlotDbWhenItsMultiAsic(t *testing.T) {
	initMultiAsic()

	mDbs, err := getAllDbsBySlot(5)
	if err != nil {
		t.Errorf("failed to get slot 5 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("slot 5 is not host db instance")
	}

	mDbs, err = getAllDbsBySlot(0)
	if err != nil {
		t.Errorf("failed to get slot 0 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("slot 0 is not host db instance")
	}

	mDbs, err = getAllDbsBySlot(4)
	if err != nil {
		t.Errorf("failed to get slot 4 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.13:6379" {
		t.Errorf("slot 4 is not host db instance")
	}

	mDbs, err = getAllDbsBySlot(3)
	if err != nil {
		t.Errorf("failed to get slot 3 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.12:6379" {
		t.Errorf("slot 3 is not host db instance")
	}

	mDbs, err = getAllDbsBySlot(2)
	if err != nil {
		t.Errorf("failed to get slot 2 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.11:6379" {
		t.Errorf("slot 2 is not host db instance")
	}

	mDbs, err = getAllDbsBySlot(1)
	if err != nil {
		t.Errorf("failed to get slot 1 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.10:6379" {
		t.Errorf("slot 1 is not host db instance")
	}
}

func TestGetSlotDbWhenItsNotMultiAsic(t *testing.T) {
	initNotMultiAsic()

	mDbs, err := getAllDbsBySlot(5)
	if err != nil {
		t.Errorf("failed to get slot 5 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("slot 5 is not host db instance")
	}

	mDbs, err = getAllDbsBySlot(0)
	if err != nil {
		t.Errorf("failed to get slot 0 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("slot 0 is not host db instance")
	}

	mDbs, err = getAllDbsBySlot(1)
	if err != nil {
		t.Errorf("failed to get slot 1 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("slot 1 is not host db instance")
	}
}

func TestDeleteDbForASlot(t *testing.T) {
	initNotMultiAsic()

	mDbs, err := getAllDbsBySlot(0)
	if err != nil {
		t.Errorf("failed to get slot 0 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("slot 0 is not host db instance")
	}

	CloseAllDbs(mDbs[:])
	if mDbs[ApplDB] != nil {
		t.Errorf("slot 0 db is not closed")
	}

	initMultiAsic()
	mDbs, err = getAllDbsBySlot(0)
	if err != nil {
		t.Errorf("failed to get slot 0 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.1:6379" {
		t.Errorf("slot 0 is not host db instance")
	}

	CloseAllDbs(mDbs[:])
	if mDbs[ApplDB] != nil {
		t.Errorf("slot 0 db is not closed")
	}

	mDbs, err = getAllDbsBySlot(4)
	if err != nil {
		t.Errorf("failed to get slot 4 db instances")
	}

	if mDbs[ApplDB].client.Options().Addr != "127.0.0.13:6379" {
		t.Errorf("slot 4 is not host db instance")
	}

	CloseAllDbs(mDbs[:])
	if mDbs[ApplDB] != nil {
		t.Errorf("slot 4 db is not closed")
	}
}

func TestDeleteDbsForAllSlots(t *testing.T) {
	initMultiAsic()

	mDbs, err := GetMDBInstances(true)
	if err != nil {
		t.Errorf("failed to get slot 1 db instances")
	}

	CloseMDBInstances(mDbs)
	if mDbs == nil{
		t.Errorf("failed to all db instances")
	}
}

//func TestCloseDbsForASlot(t *testing.T) {
//
//}

/*

1.  Create, and close a DB connection. (NewDB(), DeleteDB())

*/

func TestNewDB(t * testing.T) {

	d,e := NewDB(Options {
	                DBNo              : ConfigDB,
	                InitIndicator     : "",
	                TableNameSeparator: "|",
	                KeySeparator      : "|",
			DisableCVLCheck   : true,
                      })

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
	} else if e = d.DeleteDB() ; e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}


/*

2.  Get an entry (GetEntry())
3.  Set an entry without Transaction (SetEntry())
4.  Delete an entry without Transaction (DeleteEntry())

20. NT: GetEntry() EntryNotExist.

*/

func TestNoTransaction(t * testing.T) {

	var pid int = os.Getpid()

        d,e := NewDB(Options {
                        DBNo              : ConfigDB,
                        InitIndicator     : "",
                        TableNameSeparator: "|",
                        KeySeparator      : "|",
                        DisableCVLCheck   : true,
                      })

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec { Name: "TEST_" + strconv.FormatInt(int64(pid), 10) }

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key { Comp: ca}
	avalue := Value { map[string]string {"ports@":"Ethernet0","type":"MIRROR" }}
        e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	v, e := d.GetEntry(&ts, akey)

	if (e != nil) || (!reflect.DeepEqual(v,avalue)) {
		t.Errorf("GetEntry() fails e = %v", e)
		return
	}

        e = d.DeleteEntry(&ts, akey)

	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}

	v, e = d.GetEntry(&ts, akey)

	if e == nil {
		t.Errorf("GetEntry() after DeleteEntry() fails e = %v", e)
		return
	}

	if e = d.DeleteDB() ; e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}


/*

5.  Get a Table (GetTable())

9.  Get multiple keys (GetKeys())
10. Delete multiple keys (DeleteKeys())
11. Delete Table (DeleteTable())

*/

func TestTable(t * testing.T) {

	var pid int = os.Getpid()

        d,e := NewDB(Options {
                        DBNo              : ConfigDB,
                        InitIndicator     : "",
                        TableNameSeparator: "|",
                        KeySeparator      : "|",
                        DisableCVLCheck   : true,
                      })

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec { Name: "TEST_" + strconv.FormatInt(int64(pid), 10) }

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key { Comp: ca}
	avalue := Value { map[string]string {"ports@":"Ethernet0","type":"MIRROR" }}
	ca2 := make([]string, 1, 1)
	ca2[0] = "MyACL2_ACL_IPVNOTEXIST"
	akey2 := Key { Comp: ca2}

        // Add the Entries for Get|DeleteKeys

        e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

        e = d.SetEntry(&ts, akey2, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	keys, e := d.GetKeys(&ts)

	if (e != nil) || (len(keys) != 2) {
		t.Errorf("GetKeys() fails e = %v", e)
		return
	}

	e = d.DeleteKeys(&ts, Key {Comp: []string {"MyACL*_ACL_IPVNOTEXIST"}})

	if e != nil {
		t.Errorf("DeleteKeys() fails e = %v", e)
		return
	}

	v, e := d.GetEntry(&ts, akey)

	if e == nil {
		t.Errorf("GetEntry() after DeleteKeys() fails e = %v", e)
		return
	}



        // Add the Entries again for Table

        e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

        e = d.SetEntry(&ts, akey2, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	tab, e := d.GetTable(&ts)

	if e != nil {
		t.Errorf("GetTable() fails e = %v", e)
		return
	}

	v, e = tab.GetEntry(akey)

	if (e != nil) || (!reflect.DeepEqual(v,avalue)) {
		t.Errorf("Table.GetEntry() fails e = %v", e)
		return
	}

	e = d.DeleteTable(&ts)

	if e != nil {
		t.Errorf("DeleteTable() fails e = %v", e)
		return
	}

	v, e = d.GetEntry(&ts, akey)

	if e == nil {
		t.Errorf("GetEntry() after DeleteTable() fails e = %v", e)
		return
	}

	if e = d.DeleteDB() ; e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}


/* Tests for 

6.  Set an entry with Transaction (StartTx(), SetEntry(), CommitTx())
7.  Delete an entry with Transaction (StartTx(), DeleteEntry(), CommitTx())
8.  Abort Transaction. (StartTx(), DeleteEntry(), AbortTx())

12. Set an entry with Transaction using WatchKeys Check-And-Set(CAS)
13. Set an entry with Transaction using Table CAS
14. Set an entry with Transaction using WatchKeys, and Table CAS

15. Set an entry with Transaction with empty WatchKeys, and Table CAS
16. Negative Test(NT): Fail a Transaction using WatchKeys CAS
17. NT: Fail a Transaction using Table CAS
18. NT: Abort an Transaction with empty WatchKeys/Table CAS

Cannot Automate 19 for now
19. NT: Check V logs, Error logs

 */

func TestTransaction(t * testing.T) {
	for transRun := TransRunBasic ; transRun < TransRunEnd ; transRun++ {
		testTransaction(t, transRun)
	}
}

type TransRun int

const (
	TransRunBasic         TransRun = iota // 0
	TransRunWatchKeys                     // 1
	TransRunTable                         // 2
	TransRunWatchKeysAndTable             // 3
	TransRunEmptyWatchKeysAndTable        // 4
	TransRunFailWatchKeys                 // 5
	TransRunFailTable                     // 6

	// Nothing after this.
	TransRunEnd
)

func testTransaction(t * testing.T, transRun TransRun) {

	var pid int = os.Getpid()

        d,e := NewDB(Options {
                        DBNo              : ConfigDB,
                        InitIndicator     : "",
                        TableNameSeparator: "|",
                        KeySeparator      : "|",
                        DisableCVLCheck   : true,
                      })

	if d == nil {
		t.Errorf("NewDB() fails e = %v, transRun = %v", e, transRun)
		return
	}

	ts := TableSpec { Name: "TEST_" + strconv.FormatInt(int64(pid), 10) }

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key { Comp: ca}
	avalue := Value { map[string]string {"ports@":"Ethernet0","type":"MIRROR" }}

	var watchKeys []WatchKeys
	var table []*TableSpec

	switch transRun {
	case TransRunBasic, TransRunWatchKeysAndTable:
		watchKeys = []WatchKeys{{Ts: &ts, Key: &akey}}
		table = []*TableSpec { &ts }
	case TransRunWatchKeys, TransRunFailWatchKeys:
		watchKeys = []WatchKeys{{Ts: &ts, Key: &akey}}
		table = []*TableSpec { }
	case TransRunTable, TransRunFailTable:
		watchKeys = []WatchKeys{}
		table = []*TableSpec { &ts }
	}

	e = d.StartTx(watchKeys, table)

	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
		return
	}

        e = d.SetEntry(&ts, akey, avalue)

	if e != nil {
		t.Errorf("SetEntry() fails e = %v", e)
		return
	}

	e = d.CommitTx()

	if e != nil {
		t.Errorf("CommitTx() fails e = %v", e)
		return
	}

	v, e := d.GetEntry(&ts, akey)

	if (e != nil) || (!reflect.DeepEqual(v,avalue)) {
		t.Errorf("GetEntry() after Tx fails e = %v", e)
		return
	}

	e = d.StartTx(watchKeys, table)

	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
		return
	}

        e = d.DeleteEntry(&ts, akey)

	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}

	e = d.AbortTx()

	if e != nil {
		t.Errorf("AbortTx() fails e = %v", e)
		return
	}

	v, e = d.GetEntry(&ts, akey)

	if (e != nil) || (!reflect.DeepEqual(v,avalue)) {
		t.Errorf("GetEntry() after Abort Tx fails e = %v", e)
		return
	}

	e = d.StartTx(watchKeys, table)

	if e != nil {
		t.Errorf("StartTx() fails e = %v", e)
		return
	}

        e = d.DeleteEntry(&ts, akey)

	if e != nil {
		t.Errorf("DeleteEntry() fails e = %v", e)
		return
	}

	switch transRun {
	case TransRunFailWatchKeys, TransRunFailTable:
        	d2,_ := NewDB(Options {
                        DBNo              : ConfigDB,
                        InitIndicator     : "",
                        TableNameSeparator: "|",
                        KeySeparator      : "|",
                        DisableCVLCheck   : true,
                      })

		d2.StartTx(watchKeys, table);
        	d2.DeleteEntry(&ts, akey)
		d2.CommitTx();
		d2.DeleteDB();
	default:
	}

	e = d.CommitTx()

	switch transRun {
	case TransRunFailWatchKeys, TransRunFailTable:
		if e == nil {
			t.Errorf("NT CommitTx() tr: %v fails e = %v",
				transRun, e)
			return
		}
	default:
		if e != nil {
			t.Errorf("CommitTx() fails e = %v", e)
			return
		}
	}

	v, e = d.GetEntry(&ts, akey)

	if e == nil {
		t.Errorf("GetEntry() after Tx DeleteEntry() fails e = %v", e)
		return
	}

	d.DeleteMapAll(&ts)

	if e = d.DeleteDB() ; e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}


func TestMap(t * testing.T) {

	var pid int = os.Getpid()

	d,e := NewDB(Options {
	                DBNo              : ConfigDB,
	                InitIndicator     : "",
	                TableNameSeparator: "|",
	                KeySeparator      : "|",
			DisableCVLCheck   : true,
                      })

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec { Name: "TESTMAP_" + strconv.FormatInt(int64(pid), 10) }

	d.SetMap(&ts, "k1", "v1");
	d.SetMap(&ts, "k2", "v2");

	if v, e := d.GetMap(&ts, "k1"); v != "v1" {
		t.Errorf("GetMap() fails e = %v", e)
		return
	}

	if v, e := d.GetMapAll(&ts) ;
		(e != nil) ||
		(!reflect.DeepEqual(v,
			Value{ Field: map[string]string {
				"k1" : "v1", "k2" : "v2" }})) {
		t.Errorf("GetMapAll() fails e = %v", e)
		return
	}

	d.DeleteMapAll(&ts)

	if e = d.DeleteDB() ; e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}

func TestSubscribe(t * testing.T) {

	var pid int = os.Getpid()

	var hSetCalled, hDelCalled, delCalled bool

        d,e := NewDB(Options {
                        DBNo              : ConfigDB,
                        InitIndicator     : "",
                        TableNameSeparator: "|",
                        KeySeparator      : "|",
                        DisableCVLCheck   : true,
                      })

	if (d == nil) || (e != nil) {
		t.Errorf("NewDB() fails e = %v", e)
		return
	}

	ts := TableSpec { Name: "TEST_" + strconv.FormatInt(int64(pid), 10) }

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key { Comp: ca}
	avalue := Value { map[string]string {"ports@":"Ethernet0","type":"MIRROR" }}

	var skeys [] *SKey = make([]*SKey, 1)
        skeys[0] = & (SKey { Ts: &ts, Key: &akey,
		SEMap: map[SEvent]bool {
			SEventHSet:	true,
			SEventHDel:	true,
			SEventDel:	true,
		}})

    dbCl, _ := NewDB(Options {
		DBNo              : ConfigDB,
		InitIndicator     : "CONFIG_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator      : "|",
		DisableCVLCheck   : true,
	})
	e = SubscribeDB(dbCl, skeys, func(s *DB,
		skey *SKey, key *Key,
		event SEvent) error {
		switch event {
		case SEventHSet:
			hSetCalled = true
		case SEventHDel:
			hDelCalled = true
		case SEventDel:
			delCalled = true
		default:
		}
		return nil
	})

	if e != nil {
		t.Errorf("Subscribe() returns error e: %v", e)
		return
	}

        d.SetEntry(&ts, akey, avalue)
        d.DeleteEntryFields(&ts, akey, avalue)

	time.Sleep(5 * time.Second)

	if !hSetCalled || !hDelCalled || !delCalled {
		t.Errorf("Subscribe() callbacks missed: %v %v %v", hSetCalled,
			hDelCalled, delCalled)
		return
	}

	dbCl.UnsubscribeDB()

	time.Sleep(2 * time.Second)

	if e = d.DeleteDB() ; e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}
}
