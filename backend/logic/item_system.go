package logic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"path/filepath"
	"sync/atomic"
	"time"
)

// ... (Existing constants and vars)

var globalUIDCounter int64

func NewUID() string {
	val := atomic.AddInt64(&globalUIDCounter, 1)
	return fmt.Sprintf("ent_%d_%d", time.Now().UnixNano(), val)
}