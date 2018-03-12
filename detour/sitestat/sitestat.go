package sitestat

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
	"time"
	"log"
	"fmt"
)

type SiteStat struct {
	Update   Date                 `json:"update"`
	Vcnt     map[string]*VisitCnt `json:"site_info"` // Vcnt uses host as key
	vcntLock sync.RWMutex
}

func NewSiteStat() *SiteStat {
	return &SiteStat{
		Vcnt: map[string]*VisitCnt{},
	}
}

func (ss *SiteStat) Get(s string) *VisitCnt {
	ss.vcntLock.RLock()
	Vcnt, ok := ss.Vcnt[s]
	ss.vcntLock.RUnlock()
	if ok {
		return Vcnt
	}
	return nil
}

func (ss *SiteStat) create(s string) (vcnt *VisitCnt) {
	vcnt = newVisitCnt(0, 0)
	ss.vcntLock.Lock()
	ss.Vcnt[s] = vcnt
	ss.vcntLock.Unlock()
	return
}

func (ss *SiteStat) GetVisitCnt(host string) (vcnt *VisitCnt) {
	if vcnt = ss.Get(host); vcnt != nil {
		return
	}

	return ss.create(host)
}

func (ss *SiteStat) Store(statPath string) (err error) {
	now := time.Now()
	var savedSS *SiteStat

	savedSS = NewSiteStat()
	savedSS.Update = Date(now)
	ss.vcntLock.RLock()
	for site, vcnt := range ss.Vcnt {
		if vcnt.shouldNotSave() {
			continue
		}
		savedSS.Vcnt[site] = vcnt
	}
	ss.vcntLock.RUnlock()

	b, err := json.MarshalIndent(savedSS, "", "\t")
	if err != nil {
		log.Println("Error marshalling site stat:", err)
		panic("internal error: error marshalling site")
	}

	// Store stat into temp file first and then rename.
	// Ensures atomic update to stat file to avoid file damage.

	// Create tmp file inside config firectory to avoid cross FS rename.
	f, err := ioutil.TempFile(".", "stat")
	if err != nil {
		log.Println("create tmp file to store stat", err)
		return
	}
	if _, err = f.Write(b); err != nil {
		log.Println("Error writing stat file:", err)
		f.Close()
		return
	}
	f.Close()

	// Windows don't allow rename to existing file.
	os.Remove(statPath + ".bak")
	os.Rename(statPath, statPath+".bak")
	if err = os.Rename(f.Name(), statPath); err != nil {
		log.Println("rename new stat file", err)
		return
	}
	return
}

func (ss *SiteStat) Load(file string) (err error) {

	if file == "" {
		return
	}
	if err = isFileExists(file); err != nil {
		if !os.IsNotExist(err) {
			log.Println("Error loading stat:", err)
		}
		return
	}
	var f *os.File
	if f, err = os.Open(file); err != nil {
		log.Printf("Error opening site stat %s: %v\n", file, err)
		return
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		log.Println("Error reading site stat:", err)
		return
	}
	if err = json.Unmarshal(b, ss); err != nil {
		log.Println("Error decoding site stat:", err)
		return
	}
	return
}

func isFileExists(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() {
		return fmt.Errorf("%s is not regular file", path)
	}
	return nil
}
