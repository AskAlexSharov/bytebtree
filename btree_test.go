// Copyright 2014 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bytebtree

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

func init() {
	seed := time.Now().Unix()
	fmt.Println(seed)
	rand.Seed(seed)
}

// perm returns a random per mutation of n Int items in the range [0, n).
func perm(n int) (out [][]byte, values [][]byte) {
	for i := 0; i < n; i++ {
		key := make([]byte, 20)
		rand.Read(key)
		out = append(out, key)
		value := make([]byte, 32)
		rand.Read(value)
		values = append(values, value)
	}
	return
}

type List struct {
	keys   [][]byte
	values [][]byte
}

func (l *List) Swap(i, j int) {
	l.keys[i], l.keys[j] = l.keys[j], l.keys[i]
	l.values[i], l.values[j] = l.values[j], l.values[i]
}

func (l *List) Len() int           { return len(l.keys) }
func (l *List) Less(i, j int) bool { return bytes.Compare(l.keys[i], l.keys[j]) < 0 }

// rang returns an ordered list of Int items in the range [0, n).
func rang(n int) (keys, value [][]byte) {
	k, v := perm(n)
	l := &List{k, v}
	sort.Sort(l)
	return l.keys, l.values
}

// all extracts all items from a tree in order as a slice.
func all(t *BTree) (keys, values [][]byte) {
	t.Ascend(func(k, v []byte) bool {
		keys = append(keys, k)
		values = append(values, v)
		return true
	})
	return
}

// rangerev returns a reversed ordered list of Int items in the range [0, n).
func rangrev(n int) (keys, values [][]byte) {
	k, v := perm(n)
	l := &List{k, v}
	sort.Reverse(l)
	return l.keys, l.values
}

// allrev extracts all items from a tree in reverse order as a slice.
func allrev(t *BTree) (keys, values [][]byte) {
	t.Descend(func(k, v []byte) bool {
		keys = append(keys, k)
		values = append(values, v)
		return true
	})
	return
}

func order(keys, values [][]byte) ([][]byte, [][]byte) {
	l := &List{keys, values}
	sort.Sort(l)
	return l.keys, l.values
}
func revorder(keys, values [][]byte) ([][]byte, [][]byte) {
	l := &List{keys, values}
	sort.Reverse(l)
	return l.keys, l.values
}

func TestBTree(t *testing.T) {
	tr := New()
	const treeSize = 10_000
	for i := 0; i < 10; i++ {
		if min, _ := tr.Min(); min != nil {
			t.Fatalf("empty min, got %+v", min)
		}
		if max, _ := tr.Max(); max != nil {
			t.Fatalf("empty max, got %+v", max)
		}
		keys, values := perm(treeSize)
		for i := range keys {
			if x, _ := tr.ReplaceOrInsert(keys[i], values[i]); x != nil {
				t.Fatalf("insert found item: %x", keys[i])
			}
		}
		for i := range keys {
			if x, _ := tr.ReplaceOrInsert(keys[i], values[i]); x == nil {
				t.Fatalf("insert didn't find item, %x", keys[i])
			}
		}
		min, _ := tr.Min()
		keys, values = order(keys, values)
		if want := keys[0]; !bytes.Equal(min, want) {
			t.Fatalf("min: want %x, got %x", want, min)
		}
		max, _ := tr.Max()
		if want := keys[treeSize-1]; !bytes.Equal(max, want) {
			t.Fatalf("max: want %x, got %x", want, max)
		}
		got, gotValues := all(tr)
		want, wantValues := order(keys, values)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("mismatch:\n got: %x\nwant: %x", got, want)
		}
		if !reflect.DeepEqual(gotValues, wantValues) {
			t.Fatalf("mismatch:\n got: %x\nwant: %x", gotValues, wantValues)
		}

		gotrev, gotrevValues := all(tr)
		wantrev, wantrevValues := revorder(keys, values)
		if !reflect.DeepEqual(gotrev, wantrev) {
			t.Fatalf("mismatch:\n got: %x\nwant: %x", got, want)
		}
		if !reflect.DeepEqual(gotrevValues, wantrevValues) {
			t.Fatalf("mismatch:\n got: %x\nwant: %x", gotrevValues, wantrevValues)
		}
		for i := range keys {
			if x, _ := tr.Delete(keys[i]); x == nil {
				t.Fatalf("didn't find %v", keys[i])
			}
		}
		got, _ = all(tr)
		if len(got) > 0 {
			t.Fatalf("some left!: %v", got)
		}
	}
}

/*
func ExampleBTree() {
	tr := New(*btreeDegree)
	for i := Int(0); i < 10; i++ {
		tr.ReplaceOrInsert(i)
	}
	fmt.Println("len:       ", tr.Len())
	fmt.Println("get3:      ", tr.Get(Int(3)))
	fmt.Println("get100:    ", tr.Get(Int(100)))
	fmt.Println("del4:      ", tr.Delete(Int(4)))
	fmt.Println("del100:    ", tr.Delete(Int(100)))
	fmt.Println("replace5:  ", tr.ReplaceOrInsert(Int(5)))
	fmt.Println("replace100:", tr.ReplaceOrInsert(Int(100)))
	fmt.Println("min:       ", tr.Min())
	fmt.Println("delmin:    ", tr.DeleteMin())
	fmt.Println("max:       ", tr.Max())
	fmt.Println("delmax:    ", tr.DeleteMax())
	fmt.Println("len:       ", tr.Len())
	// Output:
	// len:        10
	// get3:       3
	// get100:     <nil>
	// del4:       4
	// del100:     <nil>
	// replace5:   5
	// replace100: <nil>
	// min:        0
	// delmin:     0
	// max:        100
	// delmax:     100
	// len:        8
}
*/

func TestDeleteMin(t *testing.T) {
	tr := New()
	keys, values := perm(100)
	for i := range keys {
		tr.ReplaceOrInsert(keys[i], values[i])
	}
	var got [][]byte
	var gotValues [][]byte
	for k, v := tr.DeleteMin(); k != nil; k, v = tr.DeleteMin() {
		got = append(got, k)
		gotValues = append(gotValues, v)
	}

	want, wantValues := order(keys, values)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ascendrange:\n got: %x\nwant: %x", got, want)
	}
	if !reflect.DeepEqual(gotValues, wantValues) {
		t.Fatalf("ascendrange:\n got: %x\nwant: %x", got, want)
	}
}

/*

func TestDeleteMax(t *testing.T) {
	tr := New(3)
	for _, v := range perm(100) {
		tr.ReplaceOrInsert(v)
	}
	var got []ItemOld
	for v := tr.DeleteMax(); v != nil; v = tr.DeleteMax() {
		got = append(got, v)
	}
	// Reverse our list.
	for i := 0; i < len(got)/2; i++ {
		got[i], got[len(got)-i-1] = got[len(got)-i-1], got[i]
	}
	if want := rang(100); !reflect.DeepEqual(got, want) {
		t.Fatalf("ascendrange:\n got: %v\nwant: %v", got, want)
	}
}

func TestAscendRange(t *testing.T) {
	tr := New(2)
	for _, v := range perm(100) {
		tr.ReplaceOrInsert(v)
	}
	var got []ItemOld
	tr.AscendRange(Int(40), Int(60), func(a ItemOld) bool {
		got = append(got, a)
		return true
	})
	if want := rang(100)[40:60]; !reflect.DeepEqual(got, want) {
		t.Fatalf("ascendrange:\n got: %v\nwant: %v", got, want)
	}
	got = got[:0]
	tr.AscendRange(Int(40), Int(60), func(a ItemOld) bool {
		if a.(Int) > 50 {
			return false
		}
		got = append(got, a)
		return true
	})
	if want := rang(100)[40:51]; !reflect.DeepEqual(got, want) {
		t.Fatalf("ascendrange:\n got: %v\nwant: %v", got, want)
	}
}

func TestDescendRange(t *testing.T) {
	tr := New(2)
	for _, v := range perm(100) {
		tr.ReplaceOrInsert(v)
	}
	var got []ItemOld
	tr.DescendRange(Int(60), Int(40), func(a ItemOld) bool {
		got = append(got, a)
		return true
	})
	if want := rangrev(100)[39:59]; !reflect.DeepEqual(got, want) {
		t.Fatalf("descendrange:\n got: %v\nwant: %v", got, want)
	}
	got = got[:0]
	tr.DescendRange(Int(60), Int(40), func(a ItemOld) bool {
		if a.(Int) < 50 {
			return false
		}
		got = append(got, a)
		return true
	})
	if want := rangrev(100)[39:50]; !reflect.DeepEqual(got, want) {
		t.Fatalf("descendrange:\n got: %v\nwant: %v", got, want)
	}
}
func TestAscendLessThan(t *testing.T) {
	tr := New(*btreeDegree)
	for _, v := range perm(100) {
		tr.ReplaceOrInsert(v)
	}
	var got []ItemOld
	tr.AscendLessThan(Int(60), func(a ItemOld) bool {
		got = append(got, a)
		return true
	})
	if want := rang(100)[:60]; !reflect.DeepEqual(got, want) {
		t.Fatalf("ascendrange:\n got: %v\nwant: %v", got, want)
	}
	got = got[:0]
	tr.AscendLessThan(Int(60), func(a ItemOld) bool {
		if a.(Int) > 50 {
			return false
		}
		got = append(got, a)
		return true
	})
	if want := rang(100)[:51]; !reflect.DeepEqual(got, want) {
		t.Fatalf("ascendrange:\n got: %v\nwant: %v", got, want)
	}
}

func TestDescendLessOrEqual(t *testing.T) {
	tr := New(*btreeDegree)
	for _, v := range perm(100) {
		tr.ReplaceOrInsert(v)
	}
	var got []ItemOld
	tr.DescendLessOrEqual(Int(40), func(a ItemOld) bool {
		got = append(got, a)
		return true
	})
	if want := rangrev(100)[59:]; !reflect.DeepEqual(got, want) {
		t.Fatalf("descendlessorequal:\n got: %v\nwant: %v", got, want)
	}
	got = got[:0]
	tr.DescendLessOrEqual(Int(60), func(a ItemOld) bool {
		if a.(Int) < 50 {
			return false
		}
		got = append(got, a)
		return true
	})
	if want := rangrev(100)[39:50]; !reflect.DeepEqual(got, want) {
		t.Fatalf("descendlessorequal:\n got: %v\nwant: %v", got, want)
	}
}
func TestAscendGreaterOrEqual(t *testing.T) {
	tr := New(*btreeDegree)
	for _, v := range perm(100) {
		tr.ReplaceOrInsert(v)
	}
	var got []ItemOld
	tr.AscendGreaterOrEqual(Int(40), func(a ItemOld) bool {
		got = append(got, a)
		return true
	})
	if want := rang(100)[40:]; !reflect.DeepEqual(got, want) {
		t.Fatalf("ascendrange:\n got: %v\nwant: %v", got, want)
	}
	got = got[:0]
	tr.AscendGreaterOrEqual(Int(40), func(a ItemOld) bool {
		if a.(Int) > 50 {
			return false
		}
		got = append(got, a)
		return true
	})
	if want := rang(100)[40:51]; !reflect.DeepEqual(got, want) {
		t.Fatalf("ascendrange:\n got: %v\nwant: %v", got, want)
	}
}

func TestDescendGreaterThan(t *testing.T) {
	tr := New(*btreeDegree)
	for _, v := range perm(100) {
		tr.ReplaceOrInsert(v)
	}
	var got []ItemOld
	tr.DescendGreaterThan(Int(40), func(a ItemOld) bool {
		got = append(got, a)
		return true
	})
	if want := rangrev(100)[:59]; !reflect.DeepEqual(got, want) {
		t.Fatalf("descendgreaterthan:\n got: %v\nwant: %v", got, want)
	}
	got = got[:0]
	tr.DescendGreaterThan(Int(40), func(a ItemOld) bool {
		if a.(Int) < 50 {
			return false
		}
		got = append(got, a)
		return true
	})
	if want := rangrev(100)[:50]; !reflect.DeepEqual(got, want) {
		t.Fatalf("descendgreaterthan:\n got: %v\nwant: %v", got, want)
	}
}

const benchmarkTreeSize = 10000

func BenchmarkInsert(b *testing.B) {
	b.StopTimer()
	insertP := perm(benchmarkTreeSize)
	b.StartTimer()
	i := 0
	for i < b.N {
		tr := New(*btreeDegree)
		for _, item := range insertP {
			tr.ReplaceOrInsert(item)
			i++
			if i >= b.N {
				return
			}
		}
	}
}

func BenchmarkSeek(b *testing.B) {
	b.StopTimer()
	size := 100000
	insertP := perm(size)
	tr := New(*btreeDegree)
	for _, item := range insertP {
		tr.ReplaceOrInsert(item)
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		tr.AscendGreaterOrEqual(Int(i%size), func(i ItemOld) bool { return false })
	}
}

func BenchmarkDeleteInsert(b *testing.B) {
	b.StopTimer()
	insertP := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, item := range insertP {
		tr.ReplaceOrInsert(item)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tr.Delete(insertP[i%benchmarkTreeSize])
		tr.ReplaceOrInsert(insertP[i%benchmarkTreeSize])
	}
}

func BenchmarkDeleteInsertCloneOnce(b *testing.B) {
	b.StopTimer()
	insertP := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, item := range insertP {
		tr.ReplaceOrInsert(item)
	}
	tr = tr.Clone()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tr.Delete(insertP[i%benchmarkTreeSize])
		tr.ReplaceOrInsert(insertP[i%benchmarkTreeSize])
	}
}

func BenchmarkDeleteInsertCloneEachTime(b *testing.B) {
	b.StopTimer()
	insertP := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, item := range insertP {
		tr.ReplaceOrInsert(item)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tr = tr.Clone()
		tr.Delete(insertP[i%benchmarkTreeSize])
		tr.ReplaceOrInsert(insertP[i%benchmarkTreeSize])
	}
}

func BenchmarkDelete(b *testing.B) {
	b.StopTimer()
	insertP := perm(benchmarkTreeSize)
	removeP := perm(benchmarkTreeSize)
	b.StartTimer()
	i := 0
	for i < b.N {
		b.StopTimer()
		tr := New(*btreeDegree)
		for _, v := range insertP {
			tr.ReplaceOrInsert(v)
		}
		b.StartTimer()
		for _, item := range removeP {
			tr.Delete(item)
			i++
			if i >= b.N {
				return
			}
		}
		if tr.Len() > 0 {
			panic(tr.Len())
		}
	}
}

func BenchmarkGet(b *testing.B) {
	b.StopTimer()
	insertP := perm(benchmarkTreeSize)
	removeP := perm(benchmarkTreeSize)
	b.StartTimer()
	i := 0
	for i < b.N {
		b.StopTimer()
		tr := New(*btreeDegree)
		for _, v := range insertP {
			tr.ReplaceOrInsert(v)
		}
		b.StartTimer()
		for _, item := range removeP {
			tr.Get(item)
			i++
			if i >= b.N {
				return
			}
		}
	}
}

func BenchmarkGetCloneEachTime(b *testing.B) {
	b.StopTimer()
	insertP := perm(benchmarkTreeSize)
	removeP := perm(benchmarkTreeSize)
	b.StartTimer()
	i := 0
	for i < b.N {
		b.StopTimer()
		tr := New(*btreeDegree)
		for _, v := range insertP {
			tr.ReplaceOrInsert(v)
		}
		b.StartTimer()
		for _, item := range removeP {
			tr = tr.Clone()
			tr.Get(item)
			i++
			if i >= b.N {
				return
			}
		}
	}
}

type byInts []ItemOld

func (a byInts) Len() int {
	return len(a)
}

func (a byInts) Less(i, j int) bool {
	return a[i].(Int) < a[j].(Int)
}

func (a byInts) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func BenchmarkAscend(b *testing.B) {
	arr := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, v := range arr {
		tr.ReplaceOrInsert(v)
	}
	sort.Sort(byInts(arr))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := 0
		tr.Ascend(func(item ItemOld) bool {
			if item.(Int) != arr[j].(Int) {
				b.Fatalf("mismatch: expected: %v, got %v", arr[j].(Int), item.(Int))
			}
			j++
			return true
		})
	}
}

func BenchmarkDescend(b *testing.B) {
	arr := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, v := range arr {
		tr.ReplaceOrInsert(v)
	}
	sort.Sort(byInts(arr))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := len(arr) - 1
		tr.Descend(func(item ItemOld) bool {
			if item.(Int) != arr[j].(Int) {
				b.Fatalf("mismatch: expected: %v, got %v", arr[j].(Int), item.(Int))
			}
			j--
			return true
		})
	}
}
func BenchmarkAscendRange(b *testing.B) {
	arr := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, v := range arr {
		tr.ReplaceOrInsert(v)
	}
	sort.Sort(byInts(arr))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := 100
		tr.AscendRange(Int(100), arr[len(arr)-100], func(item ItemOld) bool {
			if item.(Int) != arr[j].(Int) {
				b.Fatalf("mismatch: expected: %v, got %v", arr[j].(Int), item.(Int))
			}
			j++
			return true
		})
		if j != len(arr)-100 {
			b.Fatalf("expected: %v, got %v", len(arr)-100, j)
		}
	}
}

func BenchmarkDescendRange(b *testing.B) {
	arr := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, v := range arr {
		tr.ReplaceOrInsert(v)
	}
	sort.Sort(byInts(arr))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := len(arr) - 100
		tr.DescendRange(arr[len(arr)-100], Int(100), func(item ItemOld) bool {
			if item.(Int) != arr[j].(Int) {
				b.Fatalf("mismatch: expected: %v, got %v", arr[j].(Int), item.(Int))
			}
			j--
			return true
		})
		if j != 100 {
			b.Fatalf("expected: %v, got %v", len(arr)-100, j)
		}
	}
}
func BenchmarkAscendGreaterOrEqual(b *testing.B) {
	arr := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, v := range arr {
		tr.ReplaceOrInsert(v)
	}
	sort.Sort(byInts(arr))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := 100
		k := 0
		tr.AscendGreaterOrEqual(Int(100), func(item ItemOld) bool {
			if item.(Int) != arr[j].(Int) {
				b.Fatalf("mismatch: expected: %v, got %v", arr[j].(Int), item.(Int))
			}
			j++
			k++
			return true
		})
		if j != len(arr) {
			b.Fatalf("expected: %v, got %v", len(arr), j)
		}
		if k != len(arr)-100 {
			b.Fatalf("expected: %v, got %v", len(arr)-100, k)
		}
	}
}
func BenchmarkDescendLessOrEqual(b *testing.B) {
	arr := perm(benchmarkTreeSize)
	tr := New(*btreeDegree)
	for _, v := range arr {
		tr.ReplaceOrInsert(v)
	}
	sort.Sort(byInts(arr))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		j := len(arr) - 100
		k := len(arr)
		tr.DescendLessOrEqual(arr[len(arr)-100], func(item ItemOld) bool {
			if item.(Int) != arr[j].(Int) {
				b.Fatalf("mismatch: expected: %v, got %v", arr[j].(Int), item.(Int))
			}
			j--
			k--
			return true
		})
		if j != -1 {
			b.Fatalf("expected: %v, got %v", -1, j)
		}
		if k != 99 {
			b.Fatalf("expected: %v, got %v", 99, k)
		}
	}
}

const cloneTestSize = 10000

func cloneTest(t *testing.T, b *BTree, start int, p []ItemOld, wg *sync.WaitGroup, trees *[]*BTree, lock *sync.Mutex) {
	t.Logf("Starting new clone at %v", start)
	lock.Lock()
	*trees = append(*trees, b)
	lock.Unlock()
	for i := start; i < cloneTestSize; i++ {
		b.ReplaceOrInsert(p[i])
		if i%(cloneTestSize/5) == 0 {
			wg.Add(1)
			go cloneTest(t, b.Clone(), i+1, p, wg, trees, lock)
		}
	}
	wg.Done()
}

func TestCloneConcurrentOperations(t *testing.T) {
	b := New(*btreeDegree)
	trees := []*BTree{}
	p := perm(cloneTestSize)
	var wg sync.WaitGroup
	wg.Add(1)
	go cloneTest(t, b, 0, p, &wg, &trees, &sync.Mutex{})
	wg.Wait()
	want := rang(cloneTestSize)
	t.Logf("Starting equality checks on %d trees", len(trees))
	for i, tree := range trees {
		if !reflect.DeepEqual(want, all(tree)) {
			t.Errorf("tree %v mismatch", i)
		}
	}
	t.Log("Removing half from first half")
	toRemove := rang(cloneTestSize)[cloneTestSize/2:]
	for i := 0; i < len(trees)/2; i++ {
		tree := trees[i]
		wg.Add(1)
		go func() {
			for _, item := range toRemove {
				tree.Delete(item)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	t.Log("Checking all values again")
	for i, tree := range trees {
		var wantpart []ItemOld
		if i < len(trees)/2 {
			wantpart = want[:cloneTestSize/2]
		} else {
			wantpart = want
		}
		if got := all(tree); !reflect.DeepEqual(wantpart, got) {
			t.Errorf("tree %v mismatch, want %v got %v", i, len(want), len(got))
		}
	}
}

func BenchmarkDeleteAndRestore(b *testing.B) {
	items := perm(16392)
	b.ResetTimer()
	b.Run(`CopyBigFreeList`, func(b *testing.B) {
		fl := NewFreeList(16392)
		tr := NewWithFreeList(*btreeDegree, fl)
		for _, v := range items {
			tr.ReplaceOrInsert(v)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dels := make([]ItemOld, 0, tr.Len())
			tr.Ascend(ItemIterator(func(b ItemOld) bool {
				dels = append(dels, b)
				return true
			}))
			for _, del := range dels {
				tr.Delete(del)
			}
			// tr is now empty, we make a new empty copy of it.
			tr = NewWithFreeList(*btreeDegree, fl)
			for _, v := range items {
				tr.ReplaceOrInsert(v)
			}
		}
	})
	b.Run(`Copy`, func(b *testing.B) {
		tr := New(*btreeDegree)
		for _, v := range items {
			tr.ReplaceOrInsert(v)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dels := make([]ItemOld, 0, tr.Len())
			tr.Ascend(ItemIterator(func(b ItemOld) bool {
				dels = append(dels, b)
				return true
			}))
			for _, del := range dels {
				tr.Delete(del)
			}
			// tr is now empty, we make a new empty copy of it.
			tr = New(*btreeDegree)
			for _, v := range items {
				tr.ReplaceOrInsert(v)
			}
		}
	})
	b.Run(`ClearBigFreelist`, func(b *testing.B) {
		fl := NewFreeList(16392)
		tr := NewWithFreeList(*btreeDegree, fl)
		for _, v := range items {
			tr.ReplaceOrInsert(v)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tr.Clear(true)
			for _, v := range items {
				tr.ReplaceOrInsert(v)
			}
		}
	})
	b.Run(`Clear`, func(b *testing.B) {
		tr := New(*btreeDegree)
		for _, v := range items {
			tr.ReplaceOrInsert(v)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			tr.Clear(true)
			for _, v := range items {
				tr.ReplaceOrInsert(v)
			}
		}
	})
}
*/
