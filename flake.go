package flake

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"strconv"
	"sync"
	"time"
)

// 41|10|12：4096/ms
const (
	BitsStamp     = 41
	BitsMachineID = 10
	BitsSequence  = 12
	SequenceMax   = uint64(1<<63 - 1)
	SequenceMask  = uint64(1<<12 - 1)
)

// Settings 配置
type Settings struct {
	Start      uint64 `json:"start"`      // 起始值
	StartTime  string `json:"start_time"` // 起始时间
	Continuous bool   `json:"continuous"` // 是否连续，否则采用时间相关算法
	MachineID  uint64 `json:"machine_id"` // 机器/服务编号
}

// flake 雪花
type flake struct {
	sequence    uint64
	machineID   uint64
	continuous  bool
	elapsedTime uint64
	startTime   time.Time
	mutex       *sync.Mutex // 锁
}

var (
	mutex  *sync.Mutex
	flakes map[string]*flake
)

func init() {
	data, err := ioutil.ReadFile("flake.json")
	if err != nil {
		log.Fatal(err)
	}
	settings := make(map[string]Settings)
	err = json.Unmarshal(data, &settings)
	if err != nil {
		log.Fatal(err)
	}
	mutex = new(sync.Mutex)
	flakes = make(map[string]*flake)
	for k, s := range settings {
		s.init(k)
	}
}

// Start 修改起始值
func Start(name string, sn uint64) error {
	f := get(name)
	if f == nil {
		return errors.New("flake `" + name + "` NOT found")
	}
	if !f.continuous {
		return errors.New("flake `" + name + "` NOT continuous")
	}
	f.mutex.Lock()
	defer f.mutex.Unlock()

	f.sequence = sn
	return nil
}

// GenID 生成 ID
func GenID(name string) (uint64, error) {
	f := get(name)
	if f == nil {
		return 0, errors.New("flake `" + name + "` NOT found")
	}
	return f.genID()
}

// GenStrID 生成 ID
func GenStrID(name string) (string, error) {
	id, err := GenID(name)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(id, 10), nil
}

// GenHexID 生成 ID
func GenHexID(name string) (string, error) {
	id, err := GenID(name)
	if err != nil {
		return "", err
	}
	return strconv.FormatUint(id, 16), nil
}

func (s *Settings) init(name string) {
	mutex.Lock()
	defer mutex.Unlock()
	f, ok := flakes[name]
	if ok {
		return
	}
	f = &flake{
		mutex:      new(sync.Mutex),
		sequence:   s.Start,
		machineID:  s.MachineID,
		continuous: s.Continuous,
	}
	flakes[name] = f
	if f.continuous {
		return
	}

	if f.machineID >= 1024 {
		log.Fatalf("Invalid mechine id(0-1023): %d\n", f.machineID)
	}
	f.startTime = time.Unix(0, 0)
	if s.StartTime != "" {
		t, err := time.Parse("2006-01-02 15:04:05", s.StartTime)
		if err != nil || t.After(time.Now()) {
			log.Fatalf("Invalid start time setting: %s\n", s.StartTime)
		} else {
			f.startTime = t
		}
	}
	f.sequence = 0
	f.machineID = s.MachineID
}

func get(name string) *flake {
	mutex.Lock()
	defer mutex.Unlock()

	f, ok := flakes[name]
	if !ok {
		return nil
	}
	return f
}

func (f *flake) genID() (uint64, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if f.continuous {
		if f.sequence == SequenceMax {
			return 0, errors.New("OVERFLOW")
		}
		f.sequence++
		return f.sequence, nil
	}
	elapsed := f.elapsed()
	if f.elapsedTime < elapsed {
		f.elapsedTime = elapsed
		f.sequence = 0
	} else {
		f.sequence = (f.sequence + 1) & SequenceMask
		if f.sequence == 0 {
			f.elapsedTime++
		}
	}
	return f.flakeID()
}

func (f *flake) flakeID() (uint64, error) {
	if f.elapsedTime >= 1<<BitsStamp {
		return 0, errors.New("over the time limit")
	}
	return f.elapsedTime<<(BitsMachineID+BitsSequence) |
		f.machineID<<BitsMachineID |
		f.sequence, nil
}

func (f *flake) elapsed() uint64 {
	return uint64(time.Now().Sub(f.startTime).Nanoseconds() / 1e6)
}
