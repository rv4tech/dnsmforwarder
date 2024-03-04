package main

import (
	"net/netip"
	"strings"
)

func removePort(s string) string {
	i := strings.LastIndexByte(s, ':')
	if i == -1 {
		return s
	}

	s, _ = s[:i], s[i+1:]
	return s
}

func CanonAddr(a netip.Addr) netip.Addr {
	if !a.Is4In6() {
		return a
	}
	return netip.AddrFrom4(a.As4())
}

func CanonAddrFromStringSilent(s string) netip.Addr {
	s = removePort(s)
	a, _ := netip.ParseAddr(s)
	a = CanonAddr(a)
	return a
}
