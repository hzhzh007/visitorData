package visitor

import (
	"errors"
	"github.com/garyburd/redigo/redis"
	proto "github.com/golang/protobuf/proto"
	"time"
)

var (
	ErrorEmptyKey  = errors.New("empty key and return null")
	ErrorZeroValue = errors.New("zero data")
)

type RecordFlag int32

const (
	RedisVisitorPrifix = "v_"
	FlagRequestedAd    = 1 << iota
	FlagRequestedFistAdShowed
	FlagRequestedLastAdShowed
	MaxRecords = 1000
)

type VisitorData struct {
	Visitor
	uid string
}

func NewVisitor(redisConn redis.Conn, uid string) (*VisitorData, error) {
	if uid == "" {
		return nil, ErrorEmptyKey
	}
	visitorData := &VisitorData{Visitor: Visitor{Records: make([]*Record, 0)}, uid: uid}
	data, err := redis.Bytes(redisConn.Do("GET", RedisVisitorPrifix+uid))
	if err == redis.ErrNil {
		return visitorData, nil
	} else if err != nil {
		return nil, err
	}
	err = proto.Unmarshal(data, &visitorData.Visitor)
	if err != nil {
		return nil, err
	}
	return visitorData, nil
}

func (v *VisitorData) UpdateVisitorRecord(hid, vid, ttl int, flag RecordFlag, timeNow time.Time) error {
	oldest := int32(int(timeNow.Unix()) - ttl)
	for i := 0; i < len(v.Records); i++ {
		if v.Records[i].Time < oldest {
			v.Records = append(v.Records[:i], v.Records[i+1:]...) //remove the record
			continue
		}
	}
	if len(v.Records) > MaxRecords {
		v.Records = v.Records[len(v.Records)-999:]
	}
	v.Records = append(v.Records,
		&Record{
			Hid:  int32(hid),
			Vid:  int32(vid),
			Time: int32(timeNow.Unix()),
			Flag: int32(flag),
		})
	return nil
}

//save to redis, just use
func (v *VisitorData) Save(redisConn redis.Conn, ttl int) error {
	data, err := proto.Marshal(&v.Visitor)
	if len(data) == 0 {
		return ErrorZeroValue
	}
	_, err = redis.String(redisConn.Do("SET", RedisVisitorPrifix+v.uid, data, "EX", ttl))
	return err
}
