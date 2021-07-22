// Copyright 2020 Joshua J Baker. All rights reserved.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file.

package btree

// SEED=1603717878394178000 go test -run Random

import (
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var seed int64

func init() {
	var err error
	seed, err = strconv.ParseInt(os.Getenv("SEED"), 10, 64)
	if err != nil {
		seed = time.Now().UnixNano()
	}
	fmt.Printf("seed: %d\n", seed)
	rand.Seed(seed)
}

func randKeys(N int) (keys []string) {
	format := fmt.Sprintf("%%0%dd", len(fmt.Sprintf("%d", N-1)))
	for _, i := range rand.Perm(N) {
		keys = append(keys, fmt.Sprintf(format, i))
	}
	return
}

//lint:ignore U1000 used for debugging
func (tr *BTree) deepPrint() {
	tr.root.deepPrint(1)
}

func (n *node) deepPrint(level int) {
	if n == nil {
		fmt.Printf("%v %#v\n", strings.Repeat("  ", level), n)
		return
	}
	vals := fmt.Sprintf("%v ", strings.Repeat(">", level))
	vals += fmt.Sprintf("%v", n.items[:n.numItems])
	fmt.Printf("%-30s (count: %d)", vals, n.count)
	fmt.Printf("\n")
	if !n.leaf {
		for i := int16(0); i < n.numItems; i++ {
			n.children[i].deepPrint(level + 1)
		}
		n.children[n.numItems].deepPrint(level + 1)
	}
}

func stringsEquals(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type pair struct {
	key   string
	value interface{}
}

func pairLess(a, b interface{}) bool {
	return a.(pair).key < b.(pair).key
}

func TestDescend(t *testing.T) {
	tr := New(pairLess)
	var count int
	tr.Descend(pair{"1", nil}, func(item interface{}) bool {
		count++
		return true
	})
	if count > 0 {
		t.Fatalf("expected 0, got %v", count)
	}
	var keys []string
	for i := 0; i < 1000; i += 10 {
		keys = append(keys, fmt.Sprintf("%03d", i))
		tr.Set(pair{keys[len(keys)-1], nil})
	}
	var exp []string
	tr.Descend(nil, func(item interface{}) bool {
		exp = append(exp, item.(pair).key)
		return true
	})
	for i := 999; i >= 0; i-- {
		key := fmt.Sprintf("%03d", i)
		var all []string
		tr.Descend(pair{key, nil}, func(item interface{}) bool {
			all = append(all, item.(pair).key)
			return true
		})
		for len(exp) > 0 && key < exp[0] {
			exp = exp[1:]
		}
		var count int
		tr.Descend(pair{key, nil}, func(item interface{}) bool {
			if count == (i+1)%maxItems {
				return false
			}
			count++
			return true
		})
		if count > len(exp) {
			t.Fatalf("expected 1, got %v", count)
		}
		if !stringsEquals(exp, all) {
			fmt.Printf("exp: %v\n", exp)
			fmt.Printf("all: %v\n", all)
			t.Fatal("mismatch")
		}
	}
}

func TestAscend(t *testing.T) {
	tr := New(pairLess)
	var count int
	tr.Ascend(pair{"1", nil}, func(item interface{}) bool {
		count++
		return true
	})
	if count > 0 {
		t.Fatalf("expected 0, got %v", count)
	}
	var keys []string
	for i := 0; i < 1000; i += 10 {
		keys = append(keys, fmt.Sprintf("%03d", i))
		tr.Set(pair{keys[len(keys)-1], nil})
	}
	exp := keys
	for i := -1; i < 1000; i++ {
		var key string
		if i == -1 {
			key = ""
		} else {
			key = fmt.Sprintf("%03d", i)
		}
		var all []string
		tr.Ascend(pair{key, nil}, func(item interface{}) bool {
			all = append(all, item.(pair).key)
			return true
		})

		for len(exp) > 0 && key > exp[0] {
			exp = exp[1:]
		}
		var count int
		tr.Ascend(pair{key, nil}, func(item interface{}) bool {
			if count == (i+1)%maxItems {
				return false
			}
			count++
			return true
		})
		if count > len(exp) {
			t.Fatalf("expected 1, got %v", count)
		}
		if !stringsEquals(exp, all) {
			t.Fatal("mismatch")
		}
	}
}

func TestBTree(t *testing.T) {

	N := 10000
	tr := New(pairLess)
	tr.sane()
	keys := randKeys(N)

	// insert all items
	for _, key := range keys {
		item := tr.Set(pair{key, key})
		tr.sane()
		if item != nil {
			t.Fatal("expected nil")
		}
	}

	// check length
	if tr.Len() != len(keys) {
		t.Fatalf("expected %v, got %v", len(keys), tr.Len())
	}

	// get each value
	for _, key := range keys {
		item := tr.Get(pair{key, nil})
		if item == nil || item.(pair).value != key {
			t.Fatalf("expected '%v', got '%v'", key, item.(pair).value)
		}
	}

	// scan all items
	var last string
	all := make(map[string]interface{})
	tr.Ascend(nil, func(item interface{}) bool {
		key := item.(pair).key
		value := item.(pair).value
		if key <= last {
			t.Fatal("out of order")
		}
		if value.(string) != key {
			t.Fatalf("mismatch")
		}
		last = key
		all[key] = value
		return true
	})
	if len(all) != len(keys) {
		t.Fatalf("expected '%v', got '%v'", len(keys), len(all))
	}

	// reverse all items
	var prev string
	all = make(map[string]interface{})
	tr.Descend(nil, func(item interface{}) bool {
		key := item.(pair).key
		value := item.(pair).value
		if prev != "" && key >= prev {
			t.Fatal("out of order")
		}
		if value.(string) != key {
			t.Fatalf("mismatch")
		}
		prev = key
		all[key] = value
		return true
	})
	if len(all) != len(keys) {
		t.Fatalf("expected '%v', got '%v'", len(keys), len(all))
	}

	// try to get an invalid item
	item := tr.Get(pair{"invalid", nil})
	if item != nil {
		t.Fatal("expected nil")
	}

	// scan and quit at various steps
	for i := 0; i < 100; i++ {
		var j int
		tr.Ascend(nil, func(item interface{}) bool {
			if j == i {
				return false
			}
			j++
			return true
		})
	}

	// reverse and quit at various steps
	for i := 0; i < 100; i++ {
		var j int
		tr.Descend(nil, func(item interface{}) bool {
			if j == i {
				return false
			}
			j++
			return true
		})
	}

	// delete half the items
	for _, key := range keys[:len(keys)/2] {
		item := tr.Delete(pair{key, nil})
		if item == nil {
			t.Fatal("expected true")
		}
		value := item.(pair).value
		if value == nil || value.(string) != key {
			t.Fatalf("expected '%v', got '%v'", key, value)
		}
	}

	// check length
	if tr.Len() != len(keys)/2 {
		t.Fatalf("expected %v, got %v", len(keys)/2, tr.Len())
	}

	// try delete half again
	for _, key := range keys[:len(keys)/2] {
		item := tr.Delete(pair{key, nil})
		tr.sane()
		if item != nil {
			t.Fatal("expected false")
		}
	}

	// try delete half again
	for _, key := range keys[:len(keys)/2] {
		item := tr.Delete(pair{key, nil})
		tr.sane()
		if item != nil {
			t.Fatal("expected false")
		}
	}

	// check length
	if tr.Len() != len(keys)/2 {
		t.Fatalf("expected %v, got %v", len(keys)/2, tr.Len())
	}

	// scan items
	last = ""
	all = make(map[string]interface{})
	tr.Ascend(nil, func(item interface{}) bool {
		key := item.(pair).key
		value := item.(pair).value
		if key <= last {
			t.Fatal("out of order")
		}
		if value.(string) != key {
			t.Fatalf("mismatch")
		}
		last = key
		all[key] = value
		return true
	})
	if len(all) != len(keys)/2 {
		t.Fatalf("expected '%v', got '%v'", len(keys), len(all))
	}

	// replace second half
	for _, key := range keys[len(keys)/2:] {
		item := tr.Set(pair{key, key})
		tr.sane()
		if item == nil {
			t.Fatal("expected not nil")
		}
		value := item.(pair).value
		if value == nil || value.(string) != key {
			t.Fatalf("expected '%v', got '%v'", key, value)
		}
	}

	// delete next half the items
	for _, key := range keys[len(keys)/2:] {
		item := tr.Delete(pair{key, nil})
		tr.sane()
		if item == nil {
			t.Fatal("expected not nil")
		}
		value := item.(pair).value
		if value == nil || value.(string) != key {
			t.Fatalf("expected '%v', got '%v'", key, value)
		}
	}

	// check length
	if tr.Len() != 0 {
		t.Fatalf("expected %v, got %v", 0, tr.Len())
	}

	// do some stuff on an empty tree
	item = tr.Get(pair{keys[0], nil})
	if item != nil {
		t.Fatal("expected nil")
	}
	tr.Ascend(nil, func(item interface{}) bool {
		t.Fatal("should not be reached")
		return true
	})
	tr.Descend(nil, func(item interface{}) bool {
		t.Fatal("should not be reached")
		return true
	})

	item = tr.Delete(pair{"invalid", nil})
	tr.sane()
	if item != nil {
		t.Fatal("expected nil")
	}
}

func BenchmarkTidwallSequentialSet(b *testing.B) {
	tr := New(intLess)
	keys := rand.Perm(b.N)
	sort.Ints(keys)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Set(keys[i])
	}
}

func BenchmarkTidwallSequentialGet(b *testing.B) {
	tr := New(intLess)
	keys := rand.Perm(b.N)
	sort.Ints(keys)
	for i := 0; i < b.N; i++ {
		tr.Set(keys[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Get(keys[i])
	}
}

func BenchmarkTidwallRandomSet(b *testing.B) {
	tr := New(intLess)
	keys := rand.Perm(1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range keys {
			tr.Set(keys[j])
		}
	}
}

func BenchmarkTidwallRandomGet(b *testing.B) {
	tr := New(intLess)
	keys := rand.Perm(b.N)
	for i := 0; i < b.N; i++ {
		tr.Set(keys[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Get(keys[i])
	}
}

func BenchmarkTidwallSequentialSetHint(b *testing.B) {
	tr := New(intLess)
	keys := rand.Perm(b.N)
	sort.Ints(keys)
	b.ResetTimer()
	var hint PathHint
	for i := 0; i < b.N; i++ {
		tr.SetHint(keys[i], &hint)
	}
}

func BenchmarkTidwallSequentialGetHint(b *testing.B) {
	// println("\n----------------------------------------------------------------")
	tr := New(intLess)
	keys := rand.Perm(b.N)
	sort.Ints(keys)
	for i := 0; i < b.N; i++ {
		tr.Set(keys[i])
	}
	b.ResetTimer()
	var hint PathHint
	for i := 0; i < b.N; i++ {
		tr.GetHint(keys[i], &hint)
		// fmt.Printf("%064b\n", hint)
	}
}

func BenchmarkTidwallLoad(b *testing.B) {
	tr := New(intLess)
	keys := rand.Perm(b.N)
	sort.Ints(keys)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tr.Load(keys[i])
	}
}

func TestBTreeOne(t *testing.T) {
	tr := New(pairLess)
	tr.Set(pair{"1", "1"})
	tr.Delete(pair{"1", nil})
	tr.Set(pair{"1", "1"})
	tr.Delete(pair{"1", nil})
	tr.Set(pair{"1", "1"})
	tr.Delete(pair{"1", nil})
}

func TestBTree256(t *testing.T) {
	tr := New(pairLess)
	var n int
	for j := 0; j < 2; j++ {
		for _, i := range rand.Perm(256) {
			tr.Set(pair{fmt.Sprintf("%d", i), i})
			n++
			if tr.Len() != n {
				t.Fatalf("expected 256, got %d", n)
			}
		}
		for _, i := range rand.Perm(256) {
			item := tr.Get(pair{fmt.Sprintf("%d", i), nil})
			if item == nil {
				t.Fatal("expected true")
			}
			if item.(pair).value.(int) != i {
				t.Fatalf("expected %d, got %d", i, item.(pair).value.(int))
			}
		}
		for _, i := range rand.Perm(256) {
			tr.Delete(pair{fmt.Sprintf("%d", i), nil})
			n--
			if tr.Len() != n {
				t.Fatalf("expected 256, got %d", n)
			}
		}
		for _, i := range rand.Perm(256) {
			item := tr.Get(pair{fmt.Sprintf("%d", i), nil})
			if item != nil {
				t.Fatal("expected nil")
			}
		}
	}
}
func shuffle(r *rand.Rand, keys []int) {
	for i := range keys {
		j := r.Intn(i + 1)
		keys[i], keys[j] = keys[j], keys[i]
	}
}

func intLess(a, b interface{}) bool {
	return a.(int) < b.(int)
}

func TestRandom(t *testing.T) {
	r := rand.New(rand.NewSource(seed))
	N := 200000
	keys := rand.Perm(N)
	func() {
		defer func() {
			msg := fmt.Sprint(recover())
			if msg != "nil less" {
				t.Fatal("expected 'nil less' panic")
			}
		}()
		New(nil)
		t.Fatalf("reached invalid code")
	}()
	tr := New(intLess)
	checkSane := func() {
		// tr.sane()
	}
	checkSane()
	if tr.Min() != nil {
		t.Fatalf("expected nil")
	}
	if tr.Max() != nil {
		t.Fatalf("expected nil")
	}
	if tr.PopMin() != nil {
		t.Fatalf("expected nil")
	}
	if tr.PopMax() != nil {
		t.Fatalf("expected nil")
	}
	if tr.Height() != 0 {
		t.Fatalf("expected 0, got %d", tr.Height())
	}
	checkSane()
	func() {
		defer func() {
			msg := fmt.Sprint(recover())
			if msg != "nil item" {
				t.Fatal("expected 'nil item' panic")
			}
		}()
		tr.Set(nil)
		t.Fatalf("reached invalid code")
	}()
	// keys = keys[:rand.Intn(len(keys))]
	shuffle(r, keys)
	for i := 0; i < len(keys); i++ {
		prev := tr.Set(keys[i])
		checkSane()
		if prev != nil {
			t.Fatalf("expected nil")
		}
		if i%12345 == 0 {
			tr.sane()
		}
	}
	tr.sane()
	sort.Ints(keys)
	var n int
	tr.Ascend(nil, func(item interface{}) bool {
		n++
		return false
	})
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}

	n = 0
	tr.Ascend(nil, func(item interface{}) bool {
		if item != keys[n] {
			t.Fatalf("expected %d, got %d", keys[n], item)
		}
		n++
		return true
	})
	if n != len(keys) {
		t.Fatalf("expected %d, got %d", len(keys), n)
	}
	if tr.Len() != len(keys) {
		t.Fatalf("expected %d, got %d", tr.Len(), len(keys))
	}

	for i := 0; i < tr.Len(); i++ {
		item := tr.GetAt(i)
		if item != keys[i] {
			t.Fatalf("expected %d, got %d", keys[i], item)
		}
	}

	n = 0
	tr.Descend(nil, func(item interface{}) bool {
		n++
		return false
	})
	if n != 1 {
		t.Fatalf("expected 1, got %d", n)
	}
	n = 0
	tr.Descend(nil, func(item interface{}) bool {
		if item != keys[len(keys)-n-1] {
			t.Fatalf("expected %d, got %d", keys[len(keys)-n-1], item)
		}
		n++
		return true
	})
	if n != len(keys) {
		t.Fatalf("expected %d, got %d", len(keys), n)
	}
	if tr.Len() != len(keys) {
		t.Fatalf("expected %d, got %d", tr.Len(), len(keys))
	}

	checkSane()

	// tr.deepPrint()

	n = 0
	for i := 0; i < 1000; i++ {
		n := 0
		tr.Ascend(nil, func(item interface{}) bool {
			if n == i {
				return false
			}
			n++
			return true
		})
		if n != i {
			t.Fatalf("expected %d, got %d", i, n)
		}
	}

	n = 0
	for i := 0; i < 1000; i++ {
		n = 0
		tr.Descend(nil, func(item interface{}) bool {
			if n == i {
				return false
			}
			n++
			return true
		})
		if n != i {
			t.Fatalf("expected %d, got %d", i, n)
		}
	}

	sort.Ints(keys)
	for i := 0; i < len(keys); i++ {
		var res interface{}
		var j int
		tr.Ascend(keys[i], func(item interface{}) bool {
			if j == 0 {
				res = item
			}
			j++
			return j == i%500
		})
		if res != keys[i] {
			t.Fatal("not equal")
		}
	}
	for i := len(keys) - 1; i >= 0; i-- {
		var res interface{}
		var j int
		tr.Descend(keys[i], func(item interface{}) bool {
			if j == 0 {
				res = item
			}
			j++
			return j == i%500
		})
		if res != keys[i] {
			t.Fatal("not equal")
		}
	}

	if tr.Height() == 0 {
		t.Fatalf("expected non-zero")
	}
	if tr.Min() != keys[0] {
		t.Fatalf("expected '%v', got '%v'", keys[0], tr.Min())
	}
	if tr.Max() != keys[len(keys)-1] {
		t.Fatalf("expected '%v', got '%v'", keys[len(keys)-1], tr.Max())
	}
	min := tr.PopMin()
	checkSane()
	if min != keys[0] {
		t.Fatalf("expected '%v', got '%v'", keys[0], min)
	}
	max := tr.PopMax()
	checkSane()
	if max != keys[len(keys)-1] {
		t.Fatalf("expected '%v', got '%v'", keys[len(keys)-1], max)
	}
	tr.Set(min)
	tr.Set(max)
	shuffle(r, keys)
	var hint PathHint
	for i := 0; i < len(keys); i++ {
		prev := tr.Get(keys[i])
		if prev == nil || prev.(int) != keys[i] {
			t.Fatalf("expected '%v', got '%v'", keys[i], prev)
		}
		prev = tr.GetHint(keys[i], &hint)
		if prev == nil || prev.(int) != keys[i] {
			t.Fatalf("expected '%v', got '%v'", keys[i], prev)
		}
	}
	sort.Ints(keys)
	for i := 0; i < len(keys); i++ {
		item := tr.PopMin()
		if item != keys[i] {
			t.Fatalf("expected '%v', got '%v'", keys[i], item)
		}
	}
	for i := 0; i < len(keys); i++ {
		prev := tr.Set(keys[i])
		if prev != nil {
			t.Fatalf("expected nil")
		}
	}
	for i := len(keys) - 1; i >= 0; i-- {
		item := tr.PopMax()
		if item != keys[i] {
			t.Fatalf("expected '%v', got '%v'", keys[i], item)
		}
	}
	for i := 0; i < len(keys); i++ {
		prev := tr.Set(keys[i])
		if prev != nil {
			t.Fatalf("expected nil")
		}
	}

	if tr.Delete(nil) != nil {
		t.Fatal("expected nil")
	}
	checkSane()

	shuffle(r, keys)
	item := tr.Delete(keys[len(keys)/2])
	checkSane()
	if item != keys[len(keys)/2] {
		t.Fatalf("expected '%v', got '%v'", keys[len(keys)/2], item)
	}
	item2 := tr.Delete(keys[len(keys)/2])
	checkSane()
	if item2 != nil {
		t.Fatalf("expected '%v', got '%v'", nil, item2)
	}

	tr.Set(item)
	checkSane()
	for i := 0; i < len(keys); i++ {
		prev := tr.Delete(keys[i])
		checkSane()
		if prev == nil || prev.(int) != keys[i] {
			t.Fatalf("expected '%v', got '%v'", keys[i], prev)
		}
		prev = tr.Get(keys[i])
		if prev != nil {
			t.Fatalf("expected nil")
		}
		prev = tr.GetHint(keys[i], &hint)
		if prev != nil {
			t.Fatalf("expected nil")
		}
		if i%12345 == 0 {
			tr.sane()
		}
	}
	if tr.Height() != 0 {
		t.Fatalf("expected 0, got %d", tr.Height())
	}
	shuffle(r, keys)
	for i := 0; i < len(keys); i++ {
		prev := tr.Load(keys[i])
		checkSane()
		if prev != nil {
			t.Fatalf("expected nil")
		}
	}
	for i := 0; i < len(keys); i++ {
		prev := tr.Get(keys[i])
		if prev == nil || prev.(int) != keys[i] {
			t.Fatalf("expected '%v', got '%v'", keys[i], prev)
		}
	}
	shuffle(r, keys)
	for i := 0; i < len(keys); i++ {
		prev := tr.Delete(keys[i])
		checkSane()
		if prev == nil || prev.(int) != keys[i] {
			t.Fatalf("expected '%v', got '%v'", keys[i], prev)
		}
		prev = tr.Get(keys[i])
		if prev != nil {
			t.Fatalf("expected nil")
		}
	}
	sort.Ints(keys)
	for i := 0; i < len(keys); i++ {
		prev := tr.Load(keys[i])
		checkSane()
		if prev != nil {
			t.Fatalf("expected nil")
		}
	}
	shuffle(r, keys)
	item = tr.Load(keys[0])
	checkSane()
	if item != keys[0] {
		t.Fatalf("expected '%v', got '%v'", keys[0], item)
	}
	func() {
		defer func() {
			msg := fmt.Sprint(recover())
			if msg != "nil item" {
				t.Fatal("expected 'nil item' panic")
			}
		}()
		tr.Load(nil)
		checkSane()
		t.Fatalf("reached invalid code")
	}()
}

func (n *node) saneheight(height, maxheight int) bool {
	if n.leaf {
		if height != maxheight {
			return false
		}
	} else {
		i := int16(0)
		for ; i < n.numItems; i++ {
			if !n.children[i].saneheight(height+1, maxheight) {
				return false
			}
		}
		if !n.children[i].saneheight(height+1, maxheight) {
			return false
		}
	}
	return true
}

// btree_saneheight returns true if the height of all leaves match the height
// of the btree.
func (tr *BTree) saneheight() bool {
	height := tr.Height()
	if tr.root != nil {
		if height == 0 {
			return false
		}
		return tr.root.saneheight(1, height)
	}
	return height == 0
}

func (n *node) deepcount() int {
	count := int(n.numItems)
	if !n.leaf {
		for i := int16(0); i <= n.numItems; i++ {
			count += n.children[i].deepcount()
		}
	}
	if n.count != count {
		panic("!node-count")
	}
	return count
}

// btree_deepcount returns the number of items in the btree.
func (tr *BTree) deepcount() int {
	if tr.root != nil {
		return tr.root.deepcount()
	}
	return 0
}

func (tr *BTree) nodesaneprops(n *node, height int) bool {
	if height == 1 {
		if n.numItems < 1 || n.numItems > maxItems {
			return false
		}
	} else {
		if n.numItems < minItems || n.numItems > maxItems {
			return false
		}
	}
	if !n.leaf {
		for i := int16(0); i < n.numItems; i++ {
			if !tr.nodesaneprops(n.children[i], height+1) {
				return false
			}
		}
		if !tr.nodesaneprops(n.children[n.numItems], height+1) {
			return false
		}
	}
	return true
}

func (tr *BTree) saneprops() bool {
	if tr.root != nil {
		return tr.nodesaneprops(tr.root, 1)
	}
	return true
}

func (n *node) sanenils() bool {
	for i := 0; i < int(n.numItems); i++ {
		if n.items[i] == nil {
			return false
		}
	}
	for i := int(n.numItems); i < len(n.items); i++ {
		if n.items[i] != nil {
			return false
		}
	}
	if !n.leaf {
		for i := 0; i <= int(n.numItems); i++ {
			if n.children[i] == nil {
				return false
			}
			if !n.children[i].sanenils() {
				return false
			}
		}
		for i := int(n.numItems) + 1; i < len(n.children); i++ {
			if n.children[i] != nil {
				return false
			}
		}
	}
	return true
}

func (tr *BTree) sanenils() bool {
	if tr.root != nil {
		return tr.root.sanenils()
	}
	return true
}

func (tr *BTree) saneorder() bool {
	var last interface{}
	var count int
	var bad bool
	tr.Walk(func(items []interface{}) {
		for _, item := range items {
			if bad {
				return
			}
			if last != nil {
				if !tr.less(last, item) {
					bad = true
					return
				}
			}
			last = item
			count++
		}
	})
	return !bad && count == tr.count
}

// btree_sane returns true if the entire btree and every node are valid.
// - height of all leaves are the equal to the btree height.
// - deep count matches the btree count.
// - all nodes have the correct number of items and counts.
// - all items are in order.
func (tr *BTree) sane() {
	if tr == nil {
		panic("nil tree")
	}
	if !tr.saneheight() {
		panic("!sane-height")
	}
	if tr.Len() != tr.count || tr.deepcount() != tr.count {
		panic("!sane-count")
	}
	if !tr.saneprops() {
		panic("!sane-props")
	}
	if !tr.saneorder() {
		panic("!sane-order")
	}
	if !tr.sanenils() {
		panic("!sane-nils")
	}
}

type copyItem struct {
	depth int
	key   int
	value int
}

func copyItemLess(a, b interface{}) bool {
	return a.(copyItem).key < b.(copyItem).key
}

func testCopyUpdates(tr *BTree, depth, value int,
	numItems, numCopies int, maxDepth int, wg *sync.WaitGroup,
) {
	defer wg.Done()
	orgLen := tr.Len()

	var wg2 int32
	if depth < maxDepth {
		wg2 = int32(numCopies)
		wg.Add(numCopies)
		for i := 0; i < numCopies; i++ {
			go func(tr *BTree, depth, value int) {
				defer func() {
					atomic.AddInt32(&wg2, -1)
					wg.Done()
				}()

				// here we can run operations on the new tree
				// delete a bunch of items
				for _, i := range rand.Perm(numItems / 3) {
					v := tr.Delete(copyItem{key: i})
					if v.(copyItem).key != i {
						panic("invalid")
					}
				}

				// add random items with big keys
				randItems := make([]copyItem, numItems/3)
				for i, key := range rand.Perm(len(randItems)) {
					key = 100_000_000 + key
					randItems[i] = copyItem{key: key, depth: -1, value: -2}
					v := tr.Load(randItems[i])
					if v != nil {
						panic("invalid")
					}
				}

				// popmax 10
				for i := 0; i < 10; i++ {
					n := tr.Len()
					if tr.PopMax() == nil {
						if n != 0 {
							panic("invalid")
						}
					}
				}

				// popmin 10
				for i := 0; i < 10; i++ {
					n := tr.Len()
					if tr.PopMin() == nil {
						if n != 0 {
							panic("invalid")
						}
					}
				}

				// delete the random items
				for _, key := range randItems {
					tr.Delete(key)
				}

				// reset all of the items
				nvalue := rand.Int()
				for _, key := range rand.Perm(orgLen) {
					tr.Set(copyItem{key: key, value: nvalue, depth: depth})
				}

				wg.Add(1)
				go testCopyUpdates(tr, depth, nvalue,
					numItems, numCopies, maxDepth, wg)

			}(tr.Copy(), depth+1, value)
		}
	}

	// This loop runs hot, there's a gosched at the bottom to relieve pressure.
	for {
		tr.sane()
		var items []copyItem
		tr.Ascend(nil, func(item interface{}) bool {
			items = append(items, item.(copyItem))
			return true
		})
		sort.SliceIsSorted(items, func(i, j int) bool {
			return copyItemLess(items[i], items[j])
		})
		if len(items) != numItems {
			panic(fmt.Sprintf("expected '%d', got '%d'", numItems, len(items)))
		}
		if len(items) != orgLen {
			panic(fmt.Sprintf("expected '%d', got '%d'", orgLen, len(items)))
		}
		if tr.Len() != orgLen {
			panic(fmt.Sprintf("expected '%d', got '%d'", orgLen, tr.Len()))
		}
		for i, ci := range items {
			ci2 := copyItem{depth: depth, value: value, key: i}
			if ci != ci2 {
				panic(fmt.Sprintf("expected %#v, got %#v", ci2, ci))
			}
		}
		if atomic.LoadInt32(&wg2) == 0 {
			break
		}
		runtime.Gosched()
	}
}

// TestCopy tests the Copy function and performs lots of operations on a bunch
// of random btrees. This can be run with -race flag, but it may be very slow.
func TestCopy(t *testing.T) {
	start := time.Now()
	for time.Since(start) < time.Second*5 {
		maxDepth := rand.Intn(5)      // max goroutine depth
		numCopies := rand.Intn(5)     // number of copies to make per depth
		numItems := rand.Intn(10_000) // number of original items
		nvalue := rand.Int()
		tr := New(copyItemLess)
		for _, key := range rand.Perm(numItems) {
			tr.Set(copyItem{key: key, depth: 0, value: nvalue})
		}
		var wg sync.WaitGroup
		wg.Add(1)
		go testCopyUpdates(tr, 0, nvalue, numItems, numCopies, maxDepth, &wg)
		wg.Wait()
	}
}

func TestLess(t *testing.T) {
	tr := New(intLess)
	if !tr.Less(1, 2) {
		panic("invalid")
	}
	if tr.Less(2, 1) {
		panic("invalid")
	}
	if tr.Less(1, 1) {
		panic("invalid")
	}
}

func TestDeleteAt(t *testing.T) {
	N := 10_000
	tr := New(intLess)
	rand.Seed(0)
	keys := rand.Perm(N)
	for _, key := range keys {
		tr.Set(key)
	}
	tr.sane()

	for tr.Len() > 0 {
		index := rand.Intn(tr.Len())
		item1 := tr.GetAt(index)
		item2 := tr.DeleteAt(index)
		if item1 != item2 {
			panic("mismatch")
		}
		tr.sane()
	}
}
