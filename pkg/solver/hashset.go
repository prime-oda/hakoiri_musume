package solver

// HashSet is an open-addressing hash set over uint64 keys with linear probing.
// Slot value 0 represents "empty"; presence of the literal key 0 is tracked separately
// (in practice Zobrist hashes are uniformly random, so the zero key is astronomically unlikely).
//
// Memory cost is ~16 bytes per element at the 0.5 target load factor — substantially less
// than Go's map[uint64]struct{}, which carries ~50+ bytes of bucket/pointer overhead per entry.
type HashSet struct {
	slots   []uint64
	mask    uint64
	size    int
	hasZero bool
}

// NewHashSet creates a hash set sized to hold at least `expected` elements at ~0.5 load.
func NewHashSet(expected int) *HashSet {
	capacity := 16
	target := expected * 2
	for capacity < target {
		capacity <<= 1
	}
	return &HashSet{
		slots: make([]uint64, capacity),
		mask:  uint64(capacity - 1),
	}
}

// Add inserts h. Returns true if the key was newly added, false if already present.
func (s *HashSet) Add(h uint64) bool {
	if h == 0 {
		if s.hasZero {
			return false
		}
		s.hasZero = true
		s.size++
		return true
	}
	idx := h & s.mask
	for {
		slot := s.slots[idx]
		if slot == 0 {
			s.slots[idx] = h
			s.size++
			// Resize when load factor exceeds 0.7 to keep probe chains short.
			if s.size*10 > len(s.slots)*7 {
				s.resize()
			}
			return true
		}
		if slot == h {
			return false
		}
		idx = (idx + 1) & s.mask
	}
}

// Size returns the number of unique keys in the set.
func (s *HashSet) Size() int { return s.size }

func (s *HashSet) resize() {
	old := s.slots
	capacity := len(old) * 2
	s.slots = make([]uint64, capacity)
	s.mask = uint64(capacity - 1)
	for _, h := range old {
		if h == 0 {
			continue
		}
		idx := h & s.mask
		for s.slots[idx] != 0 {
			idx = (idx + 1) & s.mask
		}
		s.slots[idx] = h
	}
}
