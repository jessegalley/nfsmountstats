package main

import (
	"fmt"
	"log"

	"github.com/jessegalley/nfsmountstats"
)

func main() {
  // gather and calculate cache related stats for hitrate 
  mountstats, err := nfsmountstats.NewMountstats()
  if err != nil {
    log.Fatal(err)
  }

  // mountstats will contain every mount, even local filesystems that 
  // we don't care about, so first we'll grab just the NFS ones.
  nfsmounts := mountstats.GetNFSMountMap()
  
  fmt.Println("----------------Data Cache Stats--------------------------")
  fmt.Printf("%-20s\t%-9s%-9s%-9s%-9s\n", "mountpoint", "readKB", "serverKB", "directKB", "hitrate" )
  fmt.Println("----------------------------------------------------------")

  // nfsmounts is a map[string]*MountDevice keyed on the path to the 
  // local system's mountpoint for that mount. to use the nfs host
  // and export path instead, use mount.Device (commented below) 
  for mountpoint, mount := range nfsmounts {
    // fmt.Println(mount.Device)

    // nfs cache stats come from the bytes counters instead of the per-ops 
    // counters. there are different stats for the data and attribute caches 
    
    // NormalReadBytes represents the actual amount of bytes read from the 
    // fs at the application layer by actual read() syscalls 
    applicationBytesRead := mount.NFSInfo.Bytes.NormalReadBytes

    // ServerReadBytes represents the number of bytes read from the NFS server 
    serverBytesRead := mount.NFSInfo.Bytes.ServerReadBytes
    // DirectReadBytes represents the number of bytes read in O_DIRECT mode 
    // but for some reason this is almost always 0, even when doing O_DIRECT reads 
    directBytesRead := mount.NFSInfo.Bytes.DirectReadBytes

    var ratio float64
    if applicationBytesRead != 0 {
      // to calculate the data cache hitrate, we'll find the delta of bytes read 
      // at the application layer vs from the NFS sserver  
      clientBytesRead := serverBytesRead - directBytesRead
      ratio = (float64(applicationBytesRead - clientBytesRead) * 100) / float64(applicationBytesRead)
    }

    fmt.Printf("%-20s\t%-9d%-9d%-9d%-94.2f\n", mountpoint, applicationBytesRead/1024, serverBytesRead/1024, directBytesRead/1024, ratio)
  }

  fmt.Println("----------------Attribute Cache Stats---------------------")
  fmt.Printf("%-20s\t%-10s%-10s%-10s%-10s\n", "mountpoint", "vfsOpen", "inReval", "attInval", "dataReval" )
  fmt.Println("----------------------------------------------------------")

  for mountpoint, mount := range nfsmounts {

    // the number of times a file or dir was open()'d at the linux VFS layer
    vfsOpens := mount.NFSInfo.Events.VfsOpen

    // the number of times a GETATTR forced attribute revalidation from the NFS server 
    // these are basically attribute cache misses, I believe the attribute cache has 
    // a TTL as little as 3 seconds, so there is often lots of these.
    inodeReval := mount.NFSInfo.Events.InodeRevalidates
    // i'm not exactly sure of when attrInvals happen other than the above revals 
    attrInval := mount.NFSInfo.Events.AttrInvalidates
    // the number times an inode has had it's cached data thrown out 
    dataReval := mount.NFSInfo.Events.DataInvalidates

    fmt.Printf("%-20s\t%-10d%-10d%-10d%-10d\n", mountpoint, vfsOpens, inodeReval, attrInval, dataReval)
  }




}

