package main

import (
  "fmt"
)

func v4masklen2mask(masklen uint32) uint32 {
  return uint32(0xFFFFFFFF << (32 - masklen))
}

func ip4net(ip uint32, masklen uint32) uint32 {
  return ip & uint32(0xFFFFFFFF << (32 - masklen))
}

func v4long2ip(ip uint32) string {
  o1 := (ip & uint32(0xFF000000)) >> 24
  o2 := (ip & uint32(0xFF0000)) >> 16
  o3 := (ip & uint32(0xFF00)) >> 8
  o4 := ip & uint32(0xFF)

  return fmt.Sprintf("%d.%d.%d.%d", o1, o2, o3, o4)
}

