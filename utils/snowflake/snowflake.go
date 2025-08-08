package snowflake

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type Snowflake struct {
	Timestamp uint64
	WorkerID  uint64
	Increment uint64
}

const (
	timestampLength uint64 = 42                                    // 42
	timestampPos    uint64 = 64 - timestampLength                  // 20
	workerLength    uint64 = 10                                    // 10
	workerPos       uint64 = timestampPos - workerLength           // 12
	incrementLength        = 64 - (timestampLength + workerLength) // 12
)

var (
	maxWorkerValue    = uint64(math.Pow(2, float64(workerLength)) - 1)
	maxIncrementValue = uint64(math.Pow(2, float64(incrementLength)) - 1)

	maxTimestamp uint64 = uint64(math.Pow(2, float64(timestampLength))) // max possible timestamp value possible

	lastIncrement, lastTimestamp uint64
	mutex                        sync.Mutex

	workerID    uint64 = 0
	hasWorkerID bool   = false
)

func Setup(id uint64) error {
	if id > maxWorkerValue {
		return fmt.Errorf("worker ID value exceeds maximum value of [%d]", maxWorkerValue)
	} else if !hasWorkerID {
		workerID = id
		hasWorkerID = true
		return nil
	} else {
		return fmt.Errorf("worker ID for snowflake generator has been already set")
	}
}

func Generate() (uint64, error) {
	mutex.Lock()
	defer mutex.Unlock()

	timestamp := uint64(time.Now().UnixMilli())
	if timestamp == lastTimestamp {
		lastIncrement += 1
		if lastIncrement > maxIncrementValue {
			return 0, fmt.Errorf("increment overflow after increment reached %d", lastIncrement)
		}
	} else {
		lastIncrement = 0
		lastTimestamp = timestamp
	}

	return timestamp<<timestampPos | workerID<<workerPos | lastIncrement, nil
}

func Extract(snowflakeId uint64) Snowflake {
	snowflake := Snowflake{
		Timestamp: snowflakeId >> timestampPos,
		WorkerID:  (snowflakeId >> workerPos) & ((1 << workerLength) - 1),
		Increment: snowflakeId & ((1 << incrementLength) - 1),
	}

	return snowflake
}

func ExtractTimestamp(snowflakeId uint64) uint64 {
	return snowflakeId >> timestampPos
}

func Print(snowflakeId uint64) {
	snowflake := Extract(snowflakeId)
	// var realTimestamp = timestamp + timestampOffset

	fmt.Println("-----------------")
	fmt.Println("Snowflake:", snowflakeId)
	fmt.Println("Unix timestamp:", snowflake.Timestamp, "/", maxTimestamp)
	fmt.Println("Date:", time.UnixMilli(int64(snowflake.Timestamp)))
	fmt.Println("Years left:", (math.Pow(2.0, float64(timestampLength))-float64(snowflake.Timestamp))/1000/60/60/24/365)
	// fmt.Println("Real timestamp:", timestamp)
	fmt.Println("Worker:", workerID, "/", maxWorkerValue)
	fmt.Println("Increment:", snowflake.Increment, "/", maxIncrementValue)
	fmt.Println("-----------------")
}
