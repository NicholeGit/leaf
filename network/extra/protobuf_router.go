package extra

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/name5566/leaf/log"
	"github.com/name5566/leaf/util"
	"math"
	"reflect"
)

// -------------------------
// | id | protobuf message |
// -------------------------
type ProtobufRouter struct {
	littleEndian bool
	msgInfo      []*ProtobufMsgInfo
	msgID        map[reflect.Type]uint16
}

type ProtobufMsgInfo struct {
	msgType   reflect.Type
	msgRouter *util.CallRouter
}

func NewProtobufRouter() *ProtobufRouter {
	r := new(ProtobufRouter)
	r.littleEndian = false
	r.msgID = make(map[reflect.Type]uint16)
	return r
}

// It's dangerous to call the method on routing or marshaling
func (r *ProtobufRouter) SetByteOrder(littleEndian bool) {
	r.littleEndian = littleEndian
}

// It's dangerous to call the method on routing or marshaling
func (r *ProtobufRouter) Register(msg proto.Message, msgRouter *util.CallRouter) {
	if len(r.msgInfo) >= math.MaxUint16 {
		log.Fatal("too many protobuf messages (max = %v)", math.MaxUint16)
	}

	msgType := reflect.TypeOf(msg)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		log.Fatal("protobuf message pointer required")
	}

	i := new(ProtobufMsgInfo)
	i.msgType = msgType
	i.msgRouter = msgRouter
	r.msgInfo = append(r.msgInfo, i)

	r.msgID[msgType] = uint16(len(r.msgInfo) - 1)
}

// goroutine safe
func (r *ProtobufRouter) Route(data []byte) error {
	if len(data) < 2 {
		return errors.New("protobuf data too short")
	}

	// id
	var id uint16
	if r.littleEndian {
		id = binary.LittleEndian.Uint16(data)
	} else {
		id = binary.BigEndian.Uint16(data)
	}

	// msg
	if id >= uint16(len(r.msgInfo)) {
		return errors.New(fmt.Sprintf("message id %v not registered", id))
	}
	i := r.msgInfo[id]
	msg := reflect.New(i.msgType.Elem()).Interface().(proto.Message)
	err := proto.UnmarshalMerge(data[2:], msg)
	if err != nil {
		return err
	}

	// route
	if i.msgRouter != nil {
		i.msgRouter.Call0(i.msgType, msg)
	}

	return nil
}

// goroutine safe
func (r *ProtobufRouter) Marshal(msg proto.Message) (id uint16, data []byte, err error) {
	msgType := reflect.TypeOf(msg)

	// id
	if _id, ok := r.msgID[msgType]; !ok {
		err = errors.New(fmt.Sprintf("message %s not registered", msgType))
		return
	} else {
		id = _id
	}

	// data
	data, err = proto.Marshal(msg)
	return
}
