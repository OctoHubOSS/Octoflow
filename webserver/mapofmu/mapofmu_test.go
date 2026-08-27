//  Copyright (C) 2026 NodeByte LTD

package mapofmu

import (
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestM(t *testing.T) {

	r := rand.New(rand.NewSource(42))

	m := New[string]()
	_ = m

	keyCount := 20
	iCount := 10000
	out := make(chan string, iCount*2)

	var wg sync.WaitGroup
	wg.Add(iCount)
	for i := 0; i < iCount; i++ {
		go func(rn int) {
			defer wg.Done()
			key := strconv.Itoa(rn)

			l := m.Lock(key)
			defer l.Unlock()

			out <- key + " A"
			time.Sleep(time.Microsecond)
			out <- key + " B"
		}(r.Intn(keyCount))
	}
	wg.Wait()
	close(out)

	if l := len(m.ma); l != 0 {
		t.Errorf("unexpected map length at test end: %v", l)
	}

	outLists := make([][]string, keyCount)
	for s := range out {
		sParts := strings.Fields(s)
		kn, err := strconv.Atoi(sParts[0])
		if err != nil {
			t.Fatal(err)
		}
		outLists[kn] = append(outLists[kn], sParts[1])
	}
	for kn := 0; kn < keyCount; kn++ {
		l := outLists[kn]
		for i := 0; i < len(l); i += 2 {
			if l[i] != "A" || l[i+1] != "B" {
				t.Errorf("For key=%v and i=%v got unexpected values %v and %v", kn, i, l[i], l[i+1])
				break
			}
		}
	}
	if t.Failed() {
		t.Logf("Failed, outLists: %#v", outLists)
	}

}

func BenchmarkM(b *testing.B) {

	m := New[int]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Lock(i).Unlock()
	}

}
