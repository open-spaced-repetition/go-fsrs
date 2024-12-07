package fsrs

import (
	"math"
	"strconv"
	"time"
)

type state struct {
	C  float64
	S0 float64
	S1 float64
	S2 float64
}

type alea struct {
	c  float64
	s0 float64
	s1 float64
	s2 float64
}

func NewAlea(seed interface{}) *alea {
	mash := Mash()
	a := &alea{
		c:  1,
		s0: mash(" "),
		s1: mash(" "),
		s2: mash(" "),
	}

	if seed == nil {
		seed = time.Now().UnixNano()
	}

	seedStr := ""
	switch s := seed.(type) {
	case int:
		seedStr = strconv.Itoa(s)
	case string:
		seedStr = s
	}

	a.s0 -= mash(seedStr)
	if a.s0 < 0 {
		a.s0 += 1
	}
	a.s1 -= mash(seedStr)
	if a.s1 < 0 {
		a.s1 += 1
	}
	a.s2 -= mash(seedStr)
	if a.s2 < 0 {
		a.s2 += 1
	}

	return a
}

func (a *alea) Next() float64 {
	t := 2091639*a.s0 + a.c*2.3283064365386963e-10 // 2^-32
	a.s0 = a.s1
	a.s1 = a.s2
	a.s2 = t - math.Floor(t)
	a.c = math.Floor(t)
	return a.s2
}

func (a *alea) SetState(state state) {
	a.c = state.C
	a.s0 = state.S0
	a.s1 = state.S1
	a.s2 = state.S2
}

func (a *alea) GetState() state {
	return state{
		C:  a.c,
		S0: a.s0,
		S1: a.s1,
		S2: a.s2,
	}
}

func Mash() func(string) float64 {
	n := uint32(0xefc8249d)
	return func(data string) float64 {
		for i := 0; i < len(data); i++ {
			n += uint32(data[i])
			h := 0.02519603282416938 * float64(n)
			n = uint32(h)
			h -= float64(n)
			h *= float64(n)
			n = uint32(h)
			h -= float64(n)
			n += uint32(h * 0x100000000) // 2^32
		}
		return float64(n) * 2.3283064365386963e-10 // 2^-32
	}
}

type PRNG func() float64

func Alea(seed interface{}) PRNG {
	xg := NewAlea(seed)
	prng := func() float64 {
		return xg.Next()
	}

	return prng
}

func (prng PRNG) Int32() int32 {
	return int32(prng() * 0x100000000)
}

func (prng PRNG) Double() float64 {
	return prng() + float64(uint32(prng()*0x200000))*1.1102230246251565e-16 // 2^-53
}

func (prng PRNG) State(xg *alea) state {
	return xg.GetState()
}

func (prng PRNG) ImportState(xg *alea, state state) PRNG {
	xg.SetState(state)
	return prng
}
