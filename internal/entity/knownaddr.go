package entity

import (
	"fmt"
	"slices"
)

type KnownAddr struct {
	Pclntab AddrSpace

	Symbol         AddrSpace
	SymbolCoverage AddrCoverage
}

func NewKnownAddr() *KnownAddr {
	return &KnownAddr{
		Pclntab: make(map[uint64]*Addr),
		Symbol:  make(map[uint64]*Addr),
	}
}

func (f *KnownAddr) InsertPclntab(entry uint64, size uint64, fn *Function, meta GoPclntabMeta) {
	cur := Addr{
		AddrPos: AddrPos{
			Addr: entry,
			Size: size,
			Type: AddrTypeText,
		},
		Pkg:        fn.pkg,
		Function:   fn,
		SourceType: AddrSourceGoPclntab,

		Meta: meta,
	}
	f.Pclntab.Insert(&cur)
}

func (f *KnownAddr) InsertSymbol(entry uint64, size uint64, p *Package, typ AddrType, meta SymbolMeta) {
	cur := Addr{
		AddrPos: AddrPos{
			Addr: entry,
			Size: size,
			Type: typ,
		},
		Pkg:        p,
		Function:   nil, // TODO: try to find the function?
		SourceType: AddrSourceSymbol,

		Meta: meta,
	}
	if typ == AddrTypeText {
		if _, ok := f.Pclntab.Get(entry); ok {
			// pclntab always more accurate
			return
		}
	}
	f.Symbol.Insert(&cur)
}

func (f *KnownAddr) BuildSymbolCoverage() {
	f.SymbolCoverage = f.Symbol.ToDirtyCoverage()
}

func (f *KnownAddr) SymbolCovHas(entry uint64, size uint64) (AddrType, bool) {
	c, ok := slices.BinarySearchFunc(f.SymbolCoverage, &CoveragePart{Pos: AddrPos{Addr: entry}}, func(cur *CoveragePart, target *CoveragePart) int {
		if cur.Pos.Addr+cur.Pos.Size <= target.Pos.Addr {
			return -1
		}
		if cur.Pos.Addr >= target.Pos.Addr+size {
			return 1
		}
		return 0
	})
	if !ok {
		return "", false
	}

	return f.SymbolCoverage[c].Pos.Type, ok
}

func (f *KnownAddr) InsertDisasm(entry uint64, size uint64, fn *Function) {
	cur := Addr{
		AddrPos: AddrPos{
			Addr: entry,
			Size: size,
			Type: AddrTypeData,
		},
		Pkg:        fn.pkg,
		Function:   fn,
		SourceType: AddrSourceDisasm,
		Meta:       nil,
	}

	// symbol coverage check
	// this exists since the linker can merge some constant
	typ, ok := f.SymbolCovHas(entry, size)
	if ok {
		if typ != AddrTypeData {
			panic(fmt.Errorf("symbol %x size %x conflict with %s", entry, size, typ))
		}
		// symbol is more accurate
		return
	}

	fn.disasm.Insert(&cur)
}