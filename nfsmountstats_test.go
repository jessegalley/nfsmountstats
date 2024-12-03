package nfsmountstats_test

import (
	"strings"
	"testing"

	"github.com/jessegalley/nfsmountstats"
	"github.com/jessegalley/nfsmountstats/internal/procfs"
	"github.com/stretchr/testify/assert"
)

func TestMakeMountstats(t *testing.T) {
  procfs.PathPrefix = "testdata"
  content, err  := procfs.ReadMountstats()
  if err != nil {
    t.Errorf("couldn't read mountstats file: %v", err)
  }

  if len(content) == 0 {
    t.Errorf("empty mountstats content read")
  }

  mounts, err := nfsmountstats.NewMountstats(string(content))
  if err != nil {
    t.Errorf("error creating new Mountstats: %v", err)
  }
  
  // Should be 38 devices in the testdata file.
  assert.Equal(t, 38, len(mounts.Devices))
}

func TestGetNfsDevices(t *testing.T) {
  procfs.PathPrefix = "testdata"
  content, err  := procfs.ReadMountstats()
  if err != nil {
    t.Errorf("couldn't read mountstats file: %v", err)
  }

  if len(content) == 0 {
    t.Errorf("empty mountstats content read")
  }

  mounts, err := nfsmountstats.NewMountstats(string(content))
  if err != nil {
    t.Errorf("error creating new Mountstats: %v", err)
  }

  nfsDevices := mounts.GetNFSDevices()
  assert.Equal(t, 10, len(nfsDevices))

  tcpCount := 0
  udpCount := 0
  rdmaCount := 0
  for _, dev := range nfsDevices {
    switch dev.NFSInfo.Transport.Protocol() {
    case "tcp":
      tcpCount++
    case "udp":
      udpCount++
    case "rdma":
      rdmaCount++
    }
  }

  assert.Equal(t, 8, tcpCount)
  assert.Equal(t, 1, udpCount)
  assert.Equal(t, 1, rdmaCount)
}

func TestGetNfsDeviceMap(t *testing.T) {
  procfs.PathPrefix = "testdata"
  content, err  := procfs.ReadMountstats()
  if err != nil {
    t.Errorf("couldn't read mountstats file: %v", err)
  }

  if len(content) == 0 {
    t.Errorf("empty mountstats content read")
  }

  mounts, err := nfsmountstats.NewMountstats(string(content))
  if err != nil {
    t.Errorf("error creating new Mountstats: %v", err)
  }

  nfsDeviceMap := mounts.GetNFSMountMap()
  assert.Equal(t, 10, len(nfsDeviceMap))

  tcpCount := 0
  udpCount := 0
  rdmaCount := 0
  for _, dev := range nfsDeviceMap {
    // fmt.Println(mountpoint, ": ", dev.Device)
    switch dev.NFSInfo.Transport.Protocol() {
    case "tcp":
      tcpCount++
    case "udp":
      udpCount++
    case "rdma":
      rdmaCount++
    }
  }

  assert.Equal(t, 8, tcpCount)
  assert.Equal(t, 1, udpCount)
  assert.Equal(t, 1, rdmaCount)
  deviceDocs, ok := nfsDeviceMap["/mnt/nfs1/docs"]
  if !ok {
    t.Errorf("expected mount not present in map: `/mnt/nfs1/docs`")
  }
  assert.Equal(t, "10.0.2.31:/volume1/Public/docs", deviceDocs.Device)
}


func TestParseDevice(t *testing.T) {
  exampleDeviceText := `device 10.0.2.31:/volume1/Public/docs_work mounted on /mnt/nfs1/docs_work with fstype nfs4 statvers=1.1`
  
  mount := nfsmountstats.MountDevice{} 

  err := mount.Parse(exampleDeviceText)
  if err != nil {
    t.Errorf("couldn't parse test device string: %v", err)
  }

  assert.Equal(t, "10.0.2.31:/volume1/Public/docs_work", mount.Device)
  assert.Equal(t, "/mnt/nfs1/docs_work", mount.Mountpoint)
  assert.Equal(t, "nfs4", mount.MountType)
}

func TestParseNFSInfo(t *testing.T) {
  exampleDeviceText := ` opts:   rw,vers=4.2,rsize=1048576,wsize=1048576,namlen=255,acregmin=3,acregmax=60,acdirmin=30,acdirmax=60,hard,proto=tcp,timeo=600,retrans=2,sec=sys,clientaddr=10.0.6.15,local_lock=no
ne                                     
        age:    247663                 
        impl_id:        name='',domain='',date='0,0'
        caps:   caps=0xfffbc0b7,wtmult=512,dtsize=1048576,bsize=0,namlen=255
        nfsv4:  bm0=0xfdffafff,bm1=0xf9be3e,bm2=0x60800,acl=0x0,sessions,pnfs=not configured,lease_time=90,lease_expired=0
        sec:    flavor=1,pseudoflavor=1
        events: 10432 443365 372 1673 6485 2502 561227 206063 0 409 0 589 12831 232 9793 90 0 9695 0 10 205921 0 0 0 0 0 0 
        bytes:  109502449 121343899 0 0 10952332 121346572 2910 29875 
        RPC iostats version: 1.1  p/v: 100003/4 (nfs)  
        xprt:   tcp 0 0 60 0 10 28063 28031 3 653569 0 31 9925 9193  
        `

  nfsinfo, err := nfsmountstats.NewNFSInfo(exampleDeviceText)
  if err != nil {
    t.Errorf("failed to create new NFSInfo: %v", err)
  }
  
  assert.Equal(t, uint64(247663), nfsinfo.Age)
  assert.IsType(t, nfsmountstats.NFSEventCounters{}, nfsinfo.Events)
  assert.IsType(t, nfsmountstats.NFSByteCounters{}, nfsinfo.Bytes)
}

func TestParseNFSEvents(t *testing.T) {
  egEventsText := ` events: 10432 443365 372 1673 6485 2502 561227 206063 0 409 0 589 12831 232 9793 90 0 9695 0 10 205921 0 0 0 0 0 0 `

  events, err := nfsmountstats.NewNFSEventCounters(egEventsText)
  if err != nil {
    t.Errorf("failed to create new NFSEvents:  %v", err)
  }

  assert.Equal(t, uint64(10432), events.InodeRevalidates)
  assert.Equal(t, uint64(443365), events.DentryRevalidates)
  assert.Equal(t, uint64(372), events.DataInvalidates)
  assert.Equal(t, uint64(206063), events.VfsUpdatePage)
  assert.Equal(t, uint64(0), events.VfsReadPage)
  assert.Equal(t, uint64(0), events.Delay)
}


func TestParseNFSBytes(t *testing.T) {
  egBytesText := `bytes:  109502449 121343899 0 0 10952332 121346572 2910 29875 `

  bytes, err := nfsmountstats.NewNFSByteCounters(egBytesText)
  if err != nil {
    t.Errorf("failed to create new NFSBytes: %v", err)
  }

  assert.Equal(t, uint64(109502449), bytes.NormalReadBytes)
  assert.Equal(t, uint64(121343899), bytes.NormalWriteBytes)
  assert.Equal(t, uint64(2910), bytes.ReadPages)
  assert.Equal(t, uint64(29875), bytes.WritePages)
}

func TestParseNFSXprtUDP(t *testing.T) {
  egXprtUdpText := ` xprt:	udp 840 1 1013715537 1013715535 2 18247684089 0 `

  xprt, err := nfsmountstats.ParseNFSTransportCounters(egXprtUdpText)
  if err != nil {
    t.Errorf("failed to parse NFS Transport Counters (UDP): %v", err)
  }

  if xprt.Protocol() != "udp" {
    t.Errorf("protocol mismmatch, expected `udp`, got: %v", xprt.Protocol())
  }

  xprtUdp, ok := xprt.(*nfsmountstats.NFSTransportCountersUDP)
  if !ok {
    t.Errorf("parsed xprt text did not result in UDP struct")
  }

  assert.Equal(t, uint64(840), xprtUdp.Port)
  assert.Equal(t, uint64(1), xprtUdp.BindCount)
  assert.Equal(t, uint64(1013715537), xprtUdp.RpcSends)
  assert.Equal(t, uint64(18247684089), xprtUdp.InflightSends)
  assert.Equal(t, uint64(0), xprtUdp.BacklogUtil)
}

func TestParseNFSXprtTCPv1(t *testing.T) {
  egXprtTcpText := ` xprt:	tcp 840 1 1 0 0 1013715537 1013715535 2 18247684089 0`

  xprt, err := nfsmountstats.ParseNFSTransportCounters(egXprtTcpText)
  if err != nil {
    t.Errorf("failed to parse NFS Transport Counters (TCPv1.0): %v", err)
  }

  if xprt.Protocol() != "tcp" {
    t.Errorf("protocol mismmatch, expected `tcp`, got: %v", xprt.Protocol())
  }

  xprtTcp, ok := xprt.(*nfsmountstats.NFSTransportCountersTCP)
  if !ok {
    t.Errorf("parsed xprt text did not result in TCP struct")
  }

  assert.Equal(t, uint64(840), xprtTcp.Port)
  assert.Equal(t, uint64(1), xprtTcp.BindCount)
  assert.Equal(t, uint64(1), xprtTcp.ConnectCount)
  assert.Equal(t, uint64(1013715537), xprtTcp.RpcSends)
  assert.Equal(t, uint64(1013715535), xprtTcp.RpcReceives)
  assert.Equal(t, uint64(2), xprtTcp.BadXids)
  assert.Equal(t, uint64(18247684089), xprtTcp.InflightSends)
  assert.Equal(t, uint64(0), xprtTcp.BacklogUtil)
  // these all should be 0 with statsvers 1.0 xprt
  assert.Equal(t, uint64(0), xprtTcp.MaxRPCSlots)
  assert.Equal(t, uint64(0), xprtTcp.CumSendingQueue)
  assert.Equal(t, uint64(0), xprtTcp.CumPendingQueue)
}

func TestParseNFSXprtTCPv11(t *testing.T) {
  egXprtTcpText := ` xprt:	tcp 840 1 1 0 0 1013715537 1013715535 2 18247684089 0 1417 59765520263 15660436504`

  xprt, err := nfsmountstats.ParseNFSTransportCounters(egXprtTcpText)
  if err != nil {
    t.Errorf("failed to parse NFS Transport Counters (TCPv1.1): %v", err)
  }

  if xprt.Protocol() != "tcp" {
    t.Errorf("protocol mismmatch, expected `tcp`, got: %v", xprt.Protocol())
  }

  xprtTcp, ok := xprt.(*nfsmountstats.NFSTransportCountersTCP)
  if !ok {
    t.Errorf("parsed xprt text did not result in TCP struct")
  }

  assert.Equal(t, uint64(840), xprtTcp.Port)
  assert.Equal(t, uint64(1), xprtTcp.BindCount)
  assert.Equal(t, uint64(1), xprtTcp.ConnectCount)
  assert.Equal(t, uint64(1013715537), xprtTcp.RpcSends)
  assert.Equal(t, uint64(1013715535), xprtTcp.RpcReceives)
  assert.Equal(t, uint64(2), xprtTcp.BadXids)
  assert.Equal(t, uint64(18247684089), xprtTcp.InflightSends)
  assert.Equal(t, uint64(0), xprtTcp.BacklogUtil)
  // these all should be populated with statsvers 1.1 xprt
  assert.Equal(t, uint64(1417), xprtTcp.MaxRPCSlots)
  assert.Equal(t, uint64(59765520263), xprtTcp.CumSendingQueue)
  assert.Equal(t, uint64(15660436504), xprtTcp.CumPendingQueue)
}

func TestParseNFSXprtRDMA(t *testing.T) {
  egXprtRDMAText := `  xprt:	rdma 741 1 1 0 0 1013715537 1013715535 2 0 101371553 101371553 101371550 60 61 1 1 1 0 0 `

  xprt, err := nfsmountstats.ParseNFSTransportCounters(egXprtRDMAText)
  if err != nil {
    t.Errorf("failed to parse NFS Transport Counters (rdma): %v", err)
  }

  if xprt.Protocol() != "rdma" {
    t.Errorf("protocol mismmatch, expected `rdma`, got: %v", xprt.Protocol())
  }

  xprtRdma, ok := xprt.(*nfsmountstats.NFSTransportCountersRDMA)
  if !ok {
    t.Errorf("parsed xprt text did not result in RDMA struct")
  }

  assert.Equal(t, uint64(741), xprtRdma.Port)
  assert.Equal(t, uint64(1), xprtRdma.BindCount)
  assert.Equal(t, uint64(1), xprtRdma.ConnectCount)
  assert.Equal(t, uint64(1013715537), xprtRdma.RpcSends)
  assert.Equal(t, uint64(1013715535), xprtRdma.RpcReceives)
  assert.Equal(t, uint64(2), xprtRdma.BadXids)
  assert.Equal(t, uint64(0), xprtRdma.BacklogUtil)
  assert.Equal(t, uint64(101371553), xprtRdma.ReadChunks)
  assert.Equal(t, uint64(101371553), xprtRdma.WriteChunks)
  assert.Equal(t, uint64(101371550), xprtRdma.ReplyChunks)
  assert.Equal(t, uint64(60), xprtRdma.TotalRdmaReq)
  assert.Equal(t, uint64(61), xprtRdma.TotalRdmaRep)
  assert.Equal(t, uint64(1), xprtRdma.Pullup)
  assert.Equal(t, uint64(1), xprtRdma.Fixup)
  assert.Equal(t, uint64(1), xprtRdma.Hardway)
  assert.Equal(t, uint64(0), xprtRdma.FailedMarshal)
  assert.Equal(t, uint64(0), xprtRdma.BadReply)
}

func TestParsePerOpStatsNFSv4(t *testing.T) {
  exPerOpText := `	per-op statistics
	        NULL: 1 1 0 44 24 2 3 6 0
	        READ: 484 484 0 121212 11259100 23 2152 2190 0
	       WRITE: 513 513 0 121747828 97008 260140 5367 265518 0
	      COMMIT: 9 9 0 2124 936 0 70 70 0
	        OPEN: 916 916 0 310636 251444 56 1957 2033 374
	OPEN_CONFIRM: 0 0 0 0 0 0 0 0 0
	 OPEN_NOATTR: 1401 1401 0 425752 486288 73 2792 2894 12
	OPEN_DOWNGRADE: 1 1 0 252 112 0 2 2 0
	       CLOSE: 1921 1921 0 480620 264212 97 6005 6136 32
	     SETATTR: 691 691 0 194480 177428 10 1581 1637 0
	      FSINFO: 1 1 0 184 160 0 1 1 0
	       RENEW: 0 0 0 0 0 0 0 0 0
	 SETCLIENTID: 0 0 0 0 0 0 0 0 0
	SETCLIENTID_CONFIRM: 0 0 0 0 0 0 0 0 0
	        LOCK: 12 12 0 3696 1344 0 22 22 0
	       LOCKT: 0 0 0 0 0 0 0 0 0
	       LOCKU: 12 12 0 3168 1344 0 26 27 0
	      ACCESS: 3268 3268 0 770724 536632 85 7017 7294 1
	     GETATTR: 13920 13924 0 3187904 3394844 6668 27030 34563 7
	      LOOKUP: 5109 5109 0 1274016 1197256 96 9244 9647 1728
	 LOOKUP_ROOT: 0 0 0 0 0 0 0 0 0
	      REMOVE: 419 419 0 96740 48604 4 726 758 0
	      RENAME: 217 217 0 64332 32984 12 374 391 0
	        LINK: 82 82 0 27552 24272 0 126 131 0
	     SYMLINK: 1 1 0 292 344 0 1 1 0
	      CREATE: 104 104 0 28808 34944 1 263 271 0
	    PATHCONF: 1 1 0 176 116 0 1 1 0
	      STATFS: 3 3 0 684 480 0 7 7 0
	    READLINK: 0 0 0 0 0 0 0 0 0
	     READDIR: 465 465 0 117180 348164 5 819 854 0
	 SERVER_CAPS: 5 5 0 920 860 0 7 8 0
	 DELEGRETURN: 1305 1305 0 334024 211140 285 5243 5564 0
	      GETACL: 0 0 0 0 0 0 0 0 0
	      SETACL: 0 0 0 0 0 0 0 0 0
	FS_LOCATIONS: 0 0 0 0 0 0 0 0 0
	RELEASE_LOCKOWNER: 0 0 0 0 0 0 0 0 0
	     SECINFO: 0 0 0 0 0 0 0 0 0
	FSID_PRESENT: 0 0 0 0 0 0 0 0 0
	 EXCHANGE_ID: 37 37 0 11100 3996 0 65 69 0
	CREATE_SESSION: 71 71 0 16472 6004 0 127 134 35
	DESTROY_SESSION: 35 35 0 4200 1540 0 170 171 35
	    SEQUENCE: 3962 3988 0 542368 315916 47272 55618 102974 29
	GET_LEASE_TIME: 35 35 0 5320 3920 0 63 70 0
	RECLAIM_COMPLETE: 36 36 0 5184 3168 0 59 61 0
	   LAYOUTGET: 0 0 0 0 0 0 0 0 0
	GETDEVICEINFO: 0 0 0 0 0 0 0 0 0
	LAYOUTCOMMIT: 0 0 0 0 0 0 0 0 0
	LAYOUTRETURN: 0 0 0 0 0 0 0 0 0
	SECINFO_NO_NAME: 0 0 0 0 0 0 0 0 0
	TEST_STATEID: 0 0 0 0 0 0 0 0 0
	FREE_STATEID: 12 12 0 2256 1056 0 23 23 0
	GETDEVICELIST: 0 0 0 0 0 0 0 0 0
	BIND_CONN_TO_SESSION: 0 0 0 0 0 0 0 0 0
	DESTROY_CLIENTID: 0 0 0 0 0 0 0 0 0
	        SEEK: 0 0 0 0 0 0 0 0 0
	    ALLOCATE: 0 0 0 0 0 0 0 0 0
	  DEALLOCATE: 0 0 0 0 0 0 0 0 0
	 LAYOUTSTATS: 0 0 0 0 0 0 0 0 0
	       CLONE: 0 0 0 0 0 0 0 0 0
	        COPY: 0 0 0 0 0 0 0 0 0
	OFFLOAD_CANCEL: 0 0 0 0 0 0 0 0 0
	     LOOKUPP: 0 0 0 0 0 0 0 0 0
	 LAYOUTERROR: 0 0 0 0 0 0 0 0 0
	 COPY_NOTIFY: 0 0 0 0 0 0 0 0 0
	    GETXATTR: 0 0 0 0 0 0 0 0 0
	    SETXATTR: 0 0 0 0 0 0 0 0 0
	  LISTXATTRS: 0 0 0 0 0 0 0 0 0
	 REMOVEXATTR: 0 0 0 0 0 0 0 0 0
	   READ_PLUS: 0 0 0 0 0 0 0 0 0`

  // just test parser so we want to avoid using the constructor 
  // we'll initialize the map manually 
  nfsinfo := nfsmountstats.NFSInfo{
    RPCOpStats: make(map[string]nfsmountstats.RPCOpStat),
  }

  // per op parser is expecting a slice of lines since it will recieve 
  // the slice from the info parser, who has already split it 
  err := nfsinfo.ParsePerOpStats(strings.Split(exPerOpText, "\n"))
  if err != nil {
    t.Errorf("failed to parse per-op stats (nfsv4): %v", err)
  }

  assert.Equal(t, uint64(484), nfsinfo.RPCOpStats["READ"].Operations)
  assert.Equal(t, uint64(484), nfsinfo.RPCOpStats["READ"].Transmissions)
  assert.Equal(t, uint64(11259100), nfsinfo.RPCOpStats["READ"].BytesReceived)
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["READ"].ErrStats)

  assert.Equal(t, uint64(513), nfsinfo.RPCOpStats["WRITE"].Operations)
  assert.Equal(t, uint64(513), nfsinfo.RPCOpStats["WRITE"].Transmissions)
  assert.Equal(t, uint64(121747828), nfsinfo.RPCOpStats["WRITE"].BytesSent)
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["WRITE"].ErrStats)

  // FREE_STATEID and CREATE_SESSION are two (of many) ops exclusive fo NFSv4
	// FREE_STATEID: 12 12 0 2256 1056 0 23 23 0
  assert.Equal(t, uint64(12), nfsinfo.RPCOpStats["FREE_STATEID"].Operations)
  assert.Equal(t, uint64(12), nfsinfo.RPCOpStats["FREE_STATEID"].Transmissions)
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["FREE_STATEID"].MajorTimeouts)
  assert.Equal(t, uint64(2256), nfsinfo.RPCOpStats["FREE_STATEID"].BytesSent)
  assert.Equal(t, uint64(1056), nfsinfo.RPCOpStats["FREE_STATEID"].BytesReceived)
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["FREE_STATEID"].CumQueueTime)
  assert.Equal(t, uint64(23), nfsinfo.RPCOpStats["FREE_STATEID"].CumRespTime)
  assert.Equal(t, uint64(23), nfsinfo.RPCOpStats["FREE_STATEID"].CumTotalReqTime)
  // ErrStats seems to only be in newer versions 
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["FREE_STATEID"].ErrStats)

  // CREATE_SESSION: 71 71 0 16472 6004 0 127 134 35
  assert.Equal(t, uint64(71), nfsinfo.RPCOpStats["CREATE_SESSION"].Operations)
  assert.Equal(t, uint64(71), nfsinfo.RPCOpStats["CREATE_SESSION"].Transmissions)
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["CREATE_SESSION"].MajorTimeouts)
  assert.Equal(t, uint64(16472), nfsinfo.RPCOpStats["CREATE_SESSION"].BytesSent)
  assert.Equal(t, uint64(6004), nfsinfo.RPCOpStats["CREATE_SESSION"].BytesReceived)
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["CREATE_SESSION"].CumQueueTime)
  assert.Equal(t, uint64(127), nfsinfo.RPCOpStats["CREATE_SESSION"].CumRespTime)
  assert.Equal(t, uint64(134), nfsinfo.RPCOpStats["CREATE_SESSION"].CumTotalReqTime)
  // ErrStats seems to only be in newer versions 
  assert.Equal(t, uint64(35), nfsinfo.RPCOpStats["CREATE_SESSION"].ErrStats)

  // `READDIRPLUS` and `FSSTAT` are two ops that NFSv4 shouldn't have in it's stats 
  // so we will make sure that we _did not_ parse them for some reason 
  rdirplus, ok := nfsinfo.RPCOpStats["READDIRPLUS"]
  if ok {
    t.Errorf("expected no READDIRPLUS in NFSv4 per-ops, found: %v", rdirplus)
  }

  fsstat, ok := nfsinfo.RPCOpStats["FSSTAT"]
  if ok {
    t.Errorf("expected no FSSTAT in NFSv4 per-ops, found: %v", fsstat)
  }
}

func TestParsePerOpStatsNFSv3(t *testing.T) {
  exPerOpText:=	`per-op statistics
	        NULL: 0 0 0 0 0 0 0 0
	     GETATTR: 15118791 15118791 0 1874402980 1693304592 55867 4578417 5087338
	     SETATTR: 147487 147487 0 23294772 21238128 393 79929 82440
	      LOOKUP: 8673869 8673869 0 1200244328 2131693788 25184 9017078 9304519
	      ACCESS: 12054747 12054747 0 1542866596 1446569640 30810 3203816 3414141
	    READLINK: 244392 244392 0 30304608 39533788 699 71286 75037
	        READ: 6438446 6438446 0 875628656 94672388276 470744 23447336 24093392
	       WRITE: 573159 573159 0 7704966112 91705440 10926178 7676187 18630731
	      CREATE: 128938 128938 0 21803668 36618228 363 65779 67893
	       MKDIR: 2826 2826 0 478260 802584 8 3861 3933
	     SYMLINK: 1606 1606 0 330108 456104 4 63754 63792
	       MKNOD: 0 0 0 0 0 0 0 0
	      REMOVE: 121359 121359 0 17678796 17475696 1436 85296 88480
	       RMDIR: 3349 3349 0 474388 482256 9 2826 2905
	      RENAME: 123462 123462 0 26117596 32100120 677 74920 76471
	        LINK: 0 0 0 0 0 0 0 0
	     READDIR: 6170 6170 0 888480 149136440 21 11804 11972
	 READDIRPLUS: 1362042 1362042 0 201582216 4278783472 5037 34153934 34201891
	      FSSTAT: 92469 92469 0 11170932 15534792 683 34078 38431
	      FSINFO: 2 2 0 240 328 0 0 0
	    PATHCONF: 1 1 0 120 140 0 0 0
	      COMMIT: 0 0 0 0 0 0 0 0`

  // just test parser so we want to avoid using the constructor 
  // we'll initialize the map manually 
  nfsinfo := nfsmountstats.NFSInfo{
    RPCOpStats: make(map[string]nfsmountstats.RPCOpStat),
  }

  // per op parser is expecting a slice of lines since it will recieve 
  // the slice from the info parser, who has already split it 
  err := nfsinfo.ParsePerOpStats(strings.Split(exPerOpText, "\n"))
  if err != nil {
    t.Errorf("failed to parse per-op stats (nfsv3): %v", err)
  }

  assert.Equal(t, uint64(6438446), nfsinfo.RPCOpStats["READ"].Operations)
  assert.Equal(t, uint64(6438446), nfsinfo.RPCOpStats["READ"].Transmissions)
  assert.Equal(t, uint64(94672388276), nfsinfo.RPCOpStats["READ"].BytesReceived)
  // ErrStats should always be 0 because this is nfsv3 (statvers 1) 
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["READ"].ErrStats)
  assert.Equal(t, uint64(573159), nfsinfo.RPCOpStats["WRITE"].Operations)
  assert.Equal(t, uint64(573159), nfsinfo.RPCOpStats["WRITE"].Transmissions)
  assert.Equal(t, uint64(7704966112), nfsinfo.RPCOpStats["WRITE"].BytesSent)
  // ErrStats should always be 0 because this is nfsv3 (statvers 1) 
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["WRITE"].ErrStats)

  // `READDIRPLUS` and `FSSTAT` are two ops excludive to NFSv3 
  // READDIRPLUS: 1362042 1362042 0 201582216 4278783472 5037 34153934 34201891
  assert.Equal(t, uint64(1362042), nfsinfo.RPCOpStats["READDIRPLUS"].Operations)
  assert.Equal(t, uint64(1362042), nfsinfo.RPCOpStats["READDIRPLUS"].Transmissions)
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["READDIRPLUS"].MajorTimeouts)
  assert.Equal(t, uint64(201582216), nfsinfo.RPCOpStats["READDIRPLUS"].BytesSent)
  assert.Equal(t, uint64(4278783472), nfsinfo.RPCOpStats["READDIRPLUS"].BytesReceived)
  assert.Equal(t, uint64(5037), nfsinfo.RPCOpStats["READDIRPLUS"].CumQueueTime)
  assert.Equal(t, uint64(34153934), nfsinfo.RPCOpStats["READDIRPLUS"].CumRespTime)
  assert.Equal(t, uint64(34201891), nfsinfo.RPCOpStats["READDIRPLUS"].CumTotalReqTime)

  // FSSTAT: 92469 92469 0 11170932 15534792 683 34078 38431
  assert.Equal(t, uint64(92469), nfsinfo.RPCOpStats["FSSTAT"].Operations)
  assert.Equal(t, uint64(92469), nfsinfo.RPCOpStats["FSSTAT"].Transmissions)
  assert.Equal(t, uint64(0), nfsinfo.RPCOpStats["FSSTAT"].MajorTimeouts)
  assert.Equal(t, uint64(11170932), nfsinfo.RPCOpStats["FSSTAT"].BytesSent)
  assert.Equal(t, uint64(15534792), nfsinfo.RPCOpStats["FSSTAT"].BytesReceived)
  assert.Equal(t, uint64(683), nfsinfo.RPCOpStats["FSSTAT"].CumQueueTime)
  assert.Equal(t, uint64(34078), nfsinfo.RPCOpStats["FSSTAT"].CumRespTime)
  assert.Equal(t, uint64(38431), nfsinfo.RPCOpStats["FSSTAT"].CumTotalReqTime)

  // FREE_STATEID and CREATE_SESSION are two (of many) ops exclusive to NFSv4 
  freestateid, ok := nfsinfo.RPCOpStats["FREE_STATEID"]
  if ok {
    t.Errorf("expected no FREE_STATEID in nfsv3 ops, got: %v", freestateid)
  }
  createsession, ok := nfsinfo.RPCOpStats["CREATE_SESSION"]
  if ok {
    t.Errorf("expected no CREATE_SESSION in nfsv3 ops, got: %v", createsession)
  }
}
