package snowflake

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type Snowflake struct {
	Timestamp int64
	WorkerID  int64
	Increment int64
}

const (
	timestampLength int64 = 42                                    // 42
	timestampPos          = 64 - timestampLength                  // 20
	workerLength    int64 = 10                                    // 10
	workerPos             = timestampPos - workerLength           // 12
	incrementLength       = 64 - (timestampLength + workerLength) // 12
)

var (
	maxWorkerValue    = int64(math.Pow(2, float64(workerLength)) - 1)
	maxIncrementValue = int64(math.Pow(2, float64(incrementLength)) - 1)

	maxTimestamp = int64(math.Pow(2, float64(timestampLength))) // max possible timestamp value possible

	lastIncrement, lastTimestamp int64
	mutex                        sync.Mutex

	workerID    int64 = 0
	hasWorkerID       = false
)

func Setup(id int64) error {
	if id > maxWorkerValue {
		return fmt.Errorf("worker ID value exceeds maximum value of [%d]", maxWorkerValue)
	} else if !hasWorkerID {
		workerID = id
		hasWorkerID = true
		return nil
	}

	return fmt.Errorf("worker ID for snowflake generator has been already set")
}

func Generate() (int64, error) {
	mutex.Lock()
	defer mutex.Unlock()

	timestamp := time.Now().UnixMilli()
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

func Extract(snowflakeId int64) Snowflake {
	snowflake := Snowflake{
		Timestamp: snowflakeId >> timestampPos,
		WorkerID:  (snowflakeId >> workerPos) & ((1 << workerLength) - 1),
		Increment: snowflakeId & ((1 << incrementLength) - 1),
	}

	return snowflake
}

func ExtractTimestamp(snowflakeId int64) int64 {
	return snowflakeId >> timestampPos
}

func Print(snowflakeId int64) {
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
