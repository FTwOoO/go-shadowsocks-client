package sitestat

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

//认为被block
const blockedDelta = 2
const directDelta = 2

type vcntint int8

type Date time.Time

const dateLayout = "2006-01-02"

func (d Date) MarshalJSON() ([]byte, error) {
	return []byte("\"" + time.Time(d).Format(dateLayout) + "\""), nil
}

func (d *Date) UnmarshalJSON(input []byte) error {
	if len(input) != len(dateLayout)+2 {
		return errors.New(fmt.Sprintf("unmarshaling date: invalid input %s", string(input)))
	}
	input = input[1: len(dateLayout)+1]
	t, err := time.Parse(dateLayout, string(input))
	*d = Date(t)
	return err
}

var visitLock sync.Mutex

type VisitCnt struct {
	Direct  vcntint `json:"direct"`
	Blocked vcntint `json:"block"`
}

func newVisitCnt(direct, blocked vcntint) *VisitCnt {
	return &VisitCnt{direct, blocked}
}

//要不要记录保存为白名单，很久没访问的也不需要记录
func (vc *VisitCnt) shouldNotSave() bool {
	return (vc.Blocked == 0 && vc.Direct == 0)
}

//用于判断是否走代理
func (vc *VisitCnt) AsBlocked() bool {
	return (vc.Blocked - vc.Direct) >= blockedDelta
}

//用于设置直连的TIMEOUT为最大
func (vc *VisitCnt) AsDirect() bool {
	return (vc.Blocked == 0) || (vc.Direct-vc.Blocked >= directDelta)
}


//记录直连的结果
func (vc *VisitCnt) DirectVisit() {
	// 一次成功的直连即认为没有被block
	visitLock.Lock()
	vc.Direct += 1
	visitLock.Unlock()
}

//记录走代理的结果
func (vc *VisitCnt) BlockedVisit() {
	visitLock.Lock()
	vc.Blocked += 1
	visitLock.Unlock()
}
