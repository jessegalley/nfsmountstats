package main

import (
	"fmt"
	"log"
	"github.com/jessegalley/nfsmountstats"
)

func main() {
  // simply gather the current values for read/write throughput and ops 
  // counters and print them. nothing fancy here.

  mountstats, err := nfsmountstats.NewMountstats()
  if err != nil {
    log.Fatal(err)
  }

  // mountstats will contain every mount, even local filesystems that 
  // we don't care about, so first we'll grab just the NFS ones.
  nfsmounts := mountstats.GetNFSMountMap()
  
  fmt.Printf("%-20s\t%-9s%-9s%-9s%-9s\n", "mountpoint", "readOps", "writeOps", "readKB", "writeKB")
  fmt.Println("----------------------------------------------------------")

  // nfsmounts is a map[string]*MountDevice keyed on the path to the 
  // local system's mountpoint for that mount. to use the nfs host
  // and export path instead, use mount.Device (commented below) 
  for mountpoint, mount := range nfsmounts {
    // fmt.Println(mount.Device)

    // it's important to get the bytes and ops from the "per-ops" stats 
    // as these are what's actually on the wire.
    readStats := mount.NFSInfo.RPCOpStats["READ"]
    writeStats := mount.NFSInfo.RPCOpStats["WRITE"]

    // READ and WRITE operations are generally all _file_ related ops, so 
    // it's the majority of what we care about here. you need to sum both 
    // bytes fields for each op to get total throughput
    // for metadata ops, sumt
    readBytes := readStats.BytesSent + readStats.BytesReceived
    writeBytes := writeStats.BytesSent + writeStats.BytesReceived

    // ops counters are simple
    readOps := readStats.Operations
    writeOps := writeStats.Operations

    fmt.Printf("%-20s\t%-9d%-9d%-9d%-9d\n", mountpoint, readOps, writeOps, readBytes/1024, writeBytes/1024)
  }
}

