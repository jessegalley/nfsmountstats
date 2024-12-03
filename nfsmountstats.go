package nfsmountstats

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

)

// Mountstats struct is a representation of the content in `/proc/self/mountstats`
type Mountstats struct {
  Devices []MountDevice
}

// GetNFSDevices retuns a slice of pointers to any devices which are NFS 
func (m *Mountstats) GetNFSDevices() []*MountDevice {
  var nfsdevices []*MountDevice
  for _, dev := range m.Devices {
    if dev.MountType == "nfs" || dev.MountType == "nfs4" {
      nfsdevices = append(nfsdevices, &dev)
    }
  }

  return nfsdevices
}

// GetNFSMountMap return a map of MountDevice pointers that are 
// only NFS devices. The map index is the mountpoint path.
// Calls Mountstats.GetNFSDevices() to get a slice of pointers to map.
func (m *Mountstats) GetNFSMountMap() map[string]*MountDevice {
  nfsmap := make(map[string]*MountDevice)
  for _, dev := range m.GetNFSDevices() {
    nfsmap[dev.Mountpoint] = dev 
  }

  return nfsmap
}

// NewMountstats constructs a new Mountstats struct from content, which should be 
// a string containing the content of `/proc/self/mountstats`, calls Parse, and 
// returns a pointer to the new instance. 
// Returns error if the underlying Parse() call fails.
func NewMountstats(content string) (*Mountstats, error) {
  mounts := Mountstats{} 
  err := mounts.Parse(content)
  if err != nil {
    return nil, err 
  }
  
  return &mounts, nil
}

// Parse attempts to parse a string containing all of the content in `/proc/self/mountstats`
// creating child structs as necessary and running all parsers needed for stats and counters.
// Returns an error if any of the subsequent parses fails for any reason. 
func (m *Mountstats) Parse(text string) error {
  // i don't think there is any circumstance wherein we would actually want to 
  // _append_ what is parsed to the existing Devices slice. We're always going 
  // to want to update everything 
  // i'm also not sure if this is always going to be deterministic, the slice 
  // indexes may change for some reason im unaware of, so we can't rely on the  
  // index to point to the same Device after re-parsing the mountstats file.
  // m.Devices = make([]MountDevice, 0)
  m.Devices = nil

  // since the `mountstats` file is a bunch of devices which all start with 
  // the keyword `device` we will split on that to get every block of text 
  // that represents a mounted device.
  stringSlice := strings.Split(string(text), "device")
  if len(stringSlice) <= 1 {
    return errors.New("splitting mountstats content failed, unexpected length")
  }

  for _, v := range stringSlice {
    // sometimes we get blank (only whitespace) lines after splitting 
    // we need to check for them and skip
    if strings.TrimSpace(v) == "" { continue }

    // prepend the `device` tag back onto to the string since we used it 
    // to split the whole input 
    v = "device" + v

    // try to create a new MountDevice instance with the text, which 
    // will subsequently call all of the parsers for sub fields and sub 
    // structures. A lot happens here.
    device, err := NewMountDevice(v)
    if err != nil {
      // TODO: does it make sense to fail on any error here? or log & continue?
      return fmt.Errorf("couldn't constructe new MountDevice: %v", err)
    }
    m.Devices = append(m.Devices, *device)
  }

  return nil 
}


// MountDevice represents a single mounted device as seen inside 
// `/proc/self/mountstats`
type MountDevice struct {
  Device      string  // the device being mounted  
  Mountpoint  string  // the local path to which it's mounted 
  MountType   string  // the type of the mount 
  NFSInfo     NFSInfo // a struct of NFS info for NFS types
  OtherInfo   string  // additional info for other types 

  rawContent  string  // raw string content of this mount 
}

// NewMountDevice attempts to construct a MountDevice from `content`
// which should be a string containing the `device` line as well 
// as any following addtl information up until the the next device.
// Returns a non nill error if any of the subsequent parsing actions failed.
func NewMountDevice(content string) (*MountDevice, error) {
  // device := MountDevice{rawContent: content}
  device := MountDevice{}
  err := device.Parse(content)
  if err != nil {
    return nil, err 
  }

  return &device, nil 
}

// Parse attempts to parse the mount device text into the struct fields. 
// Will also call all subsequent parsers for child structs.
// Returns non-nil err if any parsing failed.
func (d *MountDevice) Parse(text string) error {
  // d.rawContent = text
  lines := strings.Split(text, "\n")
  fields := strings.Fields(lines[0])
  if len(fields) < 8 {
    return fmt.Errorf("invalid device line, expected >8 fields, got: %d", len(fields))
  }

  if fields[0] != "device" {
    return errors.New("invalid device line, entry did not begin with `device`")
  }
  
  if fields[2] != "mounted" {
    return errors.New("invalid device line, malformed entry")
  }

  d.Device = fields[1]
  d.Mountpoint = fields[4]
  d.MountType = fields[7]
  
  // the mount has additional lines of information, for NFS (all we care about for now)
  // it means the nfs details, stats, counters etc, so we will attempt to parse all
  // of that additional info into various child structures.
  if len(lines) > 1 {
    if d.MountType == "nfs" || d.MountType == "nfs4" {
      // if the mount type is an NFS mount, we will pass all of this other data 
      // down to the appropriate struct/parser 
      nfsinfo, err := NewNFSInfo(strings.Join(lines[1:], "\n"))
      if err != nil {
        return fmt.Errorf("error creating new NFSInfo: %v", err)
      }
      d.NFSInfo = *nfsinfo
    } else {
      // if the mounttype is _not_ NFS/NFS4, but we still have extra data, just 
      // assign it to a kind of catchall "OtherInfo" field for now.
      // We don't care about this data in the context of this program (but we might later)
      // i have no idea if this ever exsits, iscsi? cephfs? maybe?  ¯\_(ツ)_/¯ 
      d.OtherInfo = strings.Join(lines[2:], "\n")
    }
  }

  return nil
}

// NFSInfo represents all of the text data that follows a device of type 
// nfs or nfs4 in `/proc/self/mountstats`
// This data is a mix of counters, fields, etc of different formats, so 
// there are many seperate data structures and parsers for it. 
// This struct should be empty (and/or ignored) for any non-NFS mount.
type NFSInfo struct {
  Opts        string 
  Age         uint64 
  Events      NFSEventCounters
  Bytes       NFSByteCounters 
  Transport   NFSTransportCounters 
  RPCOpStats  map[string]RPCOpStat
  Other       map[string]string
}

// NewNFSInfo attempts to construct a new NFSInfo from `content`
// returns non nil err if any parsing of text fails.
func NewNFSInfo (content string) (*NFSInfo, error) {
  nfsinfo := NFSInfo{
    Other: make(map[string]string),
    RPCOpStats: make(map[string]RPCOpStat),
  }

  err := nfsinfo.Parse(content)
  if err != nil {
    return nil, err
  }
  
  return &nfsinfo, nil
}

// Parse parses string `content` which should be the entire section of infov 
// following NFS devices in `/proc/self/mountstats`, not including the next 
// device.
// It attempts to parse all of the information based on their label and fields 
// doing string conversion where necessary.
// Returns an error if any of the subsequent parsing or conversion fails.
func (i *NFSInfo) Parse(content string) error {
  lines := strings.Split(content, "\n")
  if len(lines) <= 1 {
    return errors.New("empty split of lines while parsing NFSInfo content")
  }

  for idx, line := range lines {
    line = strings.TrimSpace(line)
    if line == "" { continue }
    fields := strings.Fields(line)

    // we will check the first field of the line of information and match  
    // it against fields we care aboout, parsing accordingly
    switch fields[0] {
    case "age:":
      // the age of this NFS mount 
      age, err := strconv.ParseInt(fields[1], 10, 64)
      if err != nil {
        return fmt.Errorf("failed to parse age from line: `%v` ", line)
      }
      i.Age = uint64(age)
    case "events:":
      // the high level NFS event counters 
      eventCounters, err := NewNFSEventCounters(line)
      if err != nil {
        return err 
      }
      i.Events = *eventCounters
    case "bytes:": 
      // the high levle NFS byte counters 
      // i.Bytes = line
      byteCounters, err := NewNFSByteCounters(line)
      if err != nil {
        return err 
      }
      i.Bytes = *byteCounters
    case "xprt:":
      // the transport stats 
      // this one looks a bit different than the others since it's 
      // a "parse.." and not a "New..", this is because we're 
      // assigning to an interface, the concrete type underneath  
      // depends on what protocol is being used in the transport 
      // TODO: maybe refactor this to a constructor instead of disapatcher????
      transportCounters, err := ParseNFSTransportCounters(line)
      if err != nil {
        return err
      }
      i.Transport = transportCounters
    case "opts:":
      // NFS mount options 
      // for now I think the string representation of the opts is fine 
      // since this is how it's presented in mount or fstab anyway 
      // TODO: create some data structure and parser for the opts... if required 
      i.Opts = line 
    case "per-op":
      // per-op detailed stats, if we're here it means we want to break 
      // out of the loop because we want to parse all of these seperately 
      // we'll take the current index of the lines and send that slice to 
      // the per-op parser, then break
      err := i.ParsePerOpStats(lines[idx:])
      if err != nil {
        return fmt.Errorf("couldn't parse per-op stats in NFSInfo: %v", err)
      }
      
      return nil
    default:
      // the rest of the crap will go into a map in case for some reason 
      // we end up needing to consume it later... probably just a waste 
      i.Other[fields[0]] = line
    }
    
  }

  return nil
}

type NFSEventCounters struct {
    InodeRevalidates   uint64
    DentryRevalidates  uint64
    DataInvalidates    uint64
    AttrInvalidates    uint64
    VfsOpen            uint64
    VfsLookup          uint64
    VfsPermission      uint64
    VfsUpdatePage      uint64
    VfsReadPage        uint64
    VfsReadPages       uint64
    VfsWritePage       uint64
    VfsWritePages      uint64
    VfsReaddir         uint64
    VfsSetAttr         uint64
    VfsFlush           uint64
    VfsFsync           uint64
    VfsLock            uint64
    VfsRelease         uint64
    CongestionWait     uint64
    SetAttrTrunc       uint64
    ExtendWrite        uint64
    SillyRenames       uint64
    ShortReads         uint64
    ShortWrites        uint64
    Delay              uint64
    PNFSRead           uint64 // NFS v4.1+ only 
    PNFSWrite          uint64 // NFS v4.1+ only 
}

// NewNFSEventCounters constructs a new NFSEventCounters struct from the `events:` line 
// of the NFS mount data.
// Returnss a non-nil eror if any of the parsing fails.
func NewNFSEventCounters(eventsLine string) (*NFSEventCounters, error) {
  e := NFSEventCounters{}
  err := e.ParseNFSEventCounters(eventsLine)
  if err != nil {
    return nil, err 
  }

  return &e, nil
}
// ParseNFSEventCounters parses a single line of text representing the 
// high level event counters found in the extra info following NFS devices 
// in `/proc/self/mountstats`.
// The line of text should begin with "events:" and have N uint64 counter fields.
// example: `events:	13910 536284 513 2250 9263 2889 673643 206200 0 484 0 744 18386 346 13099 147 0 12985 0 12 206057 0 0 0 0 0 0` 
func (e *NFSEventCounters) ParseNFSEventCounters(eventsLine string) error {
  eventsLine = strings.TrimSpace(eventsLine)
  fields := strings.Fields(eventsLine)
  if eventsLine == "" || len(fields) < 26 {
    return fmt.Errorf("unexpected length or empty events line. expected > 26, got: %v", len(fields))
  }
  if fields[0] != "events:" {
    return fmt.Errorf("malformed events line: expected 'events:', got: %v", fields[0])
  }
  
  parsedInts := make([]uint64, len(fields)-1)
  for i, v := range fields[1:] {
    parsedInt, err := strconv.ParseInt(v, 10, 64)
    if err != nil {
      return fmt.Errorf("couldn't parse int field of `events:` line, actual attempt: %v", v)
    }
    parsedInts[i] = uint64(parsedInt)
  }
    // assign parsed values to struct fields, assuming at least 25 fields
    e.InodeRevalidates = parsedInts[0]
    e.DentryRevalidates = parsedInts[1]
    e.DataInvalidates = parsedInts[2]
    e.AttrInvalidates = parsedInts[3]
    e.VfsOpen = parsedInts[4]
    e.VfsLookup = parsedInts[5]
    e.VfsPermission = parsedInts[6]
    e.VfsUpdatePage = parsedInts[7]
    e.VfsReadPage = parsedInts[8]
    e.VfsReadPages = parsedInts[9]
    e.VfsWritePage = parsedInts[10]
    e.VfsWritePages = parsedInts[11]
    e.VfsReaddir = parsedInts[12]
    e.VfsSetAttr = parsedInts[13]
    e.VfsFlush = parsedInts[14]
    e.VfsFsync = parsedInts[15]
    e.VfsLock = parsedInts[16]
    e.VfsRelease = parsedInts[17]
    e.CongestionWait = parsedInts[18]
    e.SetAttrTrunc = parsedInts[19]
    e.ExtendWrite = parsedInts[20]
    e.SillyRenames = parsedInts[21]
    e.ShortReads = parsedInts[22]
    e.ShortWrites = parsedInts[23]
    e.Delay = parsedInts[24]

    // check for optional PNFSRead and PNFSWrite
    if len(parsedInts) > 25 {
        e.PNFSRead = parsedInts[25]
    }
    if len(parsedInts) > 26 {
        e.PNFSWrite = parsedInts[26]
    }

  return nil
}

type NFSByteCounters struct {
    NormalReadBytes   uint64
    NormalWriteBytes  uint64
    DirectReadBytes   uint64
    DirectWriteBytes  uint64
    ServerReadBytes   uint64
    ServerWriteBytes  uint64
    ReadPages         uint64
    WritePages        uint64
}

func NewNFSByteCounters(bytesLine string) (*NFSByteCounters, error) {
  byteCounters := NFSByteCounters{}
  err := byteCounters.ParseNFSByteCounters(bytesLine)
  if err != nil {
    return nil, err 
  }

  return &byteCounters, nil
}
// ParseNFSByteCounters parses a single line of text representing the 
// byte counters found in the extra info following NFS devices 
// in `/proc/self/mountstats`.
// The line of text should begin with "bytes:" and have 8 uint64 counter fields.
// example: `bytes: 119180641567 7459840923 0 0 93848122978 7622270673 26459312 1932867`
func (b *NFSByteCounters) ParseNFSByteCounters(bytesLine string) error {
    bytesLine = strings.TrimSpace(bytesLine)
    fields := strings.Fields(bytesLine)
    
    // Check for a valid line starting with "bytes:" and containing exactly 9 fields
    if bytesLine == "" || len(fields) != 9 {
        return fmt.Errorf("unexpected length or empty bytes line. expected exactly 9 fields, got: %v", len(fields))
    }

    if fields[0] != "bytes:" {
        return fmt.Errorf("malformed bytes line: expected 'bytes:', got: %v", fields[0])
    }

    // Parse fields after "bytes:"
    parsedInts := make([]uint64, 8)
    for i, v := range fields[1:9] { // Take only the next 8 fields
        parsedInt, err := strconv.ParseUint(v, 10, 64)
        if err != nil {
            return fmt.Errorf("couldn't parse uint field of `bytes:` line, actual attempt: %v", v)
        }
        parsedInts[i] = parsedInt
    }

    // Assign parsed values to struct fields
    b.NormalReadBytes = parsedInts[0]
    b.NormalWriteBytes = parsedInts[1]
    b.DirectReadBytes = parsedInts[2]
    b.DirectWriteBytes = parsedInts[3]
    b.ServerReadBytes = parsedInts[4]
    b.ServerWriteBytes = parsedInts[5]
    b.ReadPages = parsedInts[6]
    b.WritePages = parsedInts[7]

    return nil
}

type NFSTransportCounters interface {
    ParseCounters(fields []string) error
    Protocol() string 
}

// ParseNFSTransportCounters parses a single line of text for different NFS transport protocols (TCP, UDP, RDMA).
// Based on the transport type in the 2nd field, it initializes the corresponding struct and parses the counters.
func ParseNFSTransportCounters(xprtLine string) (NFSTransportCounters, error) {
    xprtLine= strings.TrimSpace(xprtLine)
    fields := strings.Fields(xprtLine)

    if len(fields) < 3 {
        return nil, fmt.Errorf("invalid xprt line, not enough fields")
    }

    protocol := fields[1]
    var counter NFSTransportCounters

    switch protocol {
    case "udp":
        counter = &NFSTransportCountersUDP{}
    case "tcp":
        counter = &NFSTransportCountersTCP{}
    case "rdma":
        counter = &NFSTransportCountersRDMA{}
    default:
        return nil, fmt.Errorf("unsupported protocol: %s", protocol)
    }

    err := counter.ParseCounters(fields)
    if err != nil {
        return nil, err
    }

    return counter, nil
}

type NFSTransportCountersUDP struct {
    Port          uint64
    BindCount     uint64
    RpcSends      uint64
    RpcReceives   uint64
    BadXids       uint64
    InflightSends uint64
    BacklogUtil   uint64
}

func (u *NFSTransportCountersUDP) Protocol() string {
  return "udp"
}

func (u *NFSTransportCountersUDP) ParseCounters(fields []string) error {
    if fields[0] != "xprt:" {
      return fmt.Errorf("malformed xprt: line in UDP parser, expected 'xprt:', got %v", fields[0])
    }
    if fields[1] != "udp" {
      return fmt.Errorf("malformed xprt: line in UDP parser, expected 'udp', got %v", fields[1])
    }
    if len(fields) < 9 {
        return fmt.Errorf("unexpected number of fields for UDP, expected at least 9, got: %v", len(fields))
    }

    parsedInts := make([]uint64, len(fields)-2)
    for i, v := range fields[2:] {
      parsedInt, err := strconv.ParseUint(v, 10, 64)
      if err != nil {
        return fmt.Errorf("error parsing uint in udp parser, idx: %d, actual: %v (%v)", i, v, err)
      }
      parsedInts[i] = parsedInt
    }

    u.Port = parsedInts[0]
    u.BindCount = parsedInts[1]
    u.RpcSends = parsedInts[2]
    u.RpcReceives = parsedInts[3]
    u.BadXids = parsedInts[4]
    u.InflightSends = parsedInts[5]
    u.BacklogUtil = parsedInts[6]

    return nil
}

type NFSTransportCountersTCP struct {
    // statvers 1.0 (everything should have these fields)
    Port          uint64
    BindCount     uint64
    ConnectCount  uint64
    ConnectTime   uint64
    IdleTime      uint64
    RpcSends      uint64
    RpcReceives   uint64
    BadXids       uint64
    InflightSends uint64 // cumulative 'active' request count 
    BacklogUtil   uint64 // cumulative backlog request count 

    // statvers 1.1+ also has the following fields 
    // this shuld be found in "newer" linux kernels 
    MaxRPCSlots   uint64 // the maximum number of simultaneously active  
                         // rpc slots that this mount ever had 
    CumSendingQueue uint64  // Every time we send a request, we 
                            // add the current size of the sending 
                            // queue to this counter.
    CumPendingQueue uint64  // Every time we send a request, we add 
                            // the current size of the pending queue 
                            // to this counter.

}

func (u *NFSTransportCountersTCP) Protocol() string {
  return "tcp"
}

func (t *NFSTransportCountersTCP) ParseCounters(fields []string) error {
    if fields[0] != "xprt:" {
        return fmt.Errorf("malformed xprt: line in TCP parser, expected 'xprt:', got %v", fields[0])
    }
    if fields[1] != "tcp" {
        return fmt.Errorf("malformed xprt: line in TCP parser, expected 'tcp', got %v", fields[1])
    }
    // statvers 1.0 will specify a minimum of 10 counters, +2 fields for label and protocol
    // so we check for 12.
    // statvers 1.1 will have more, but we will check/assign those dynamically if they exist 
    if len(fields) < 12 {
        return fmt.Errorf("unexpected number of fields for TCP, expected at least 12, got: %v", len(fields))
    }

    // parse the string fields after "xprt: tcp"
    // dynamically size this array incase we have statvers 1.1+ extra fields 
    parsedInts := make([]uint64, len(fields)-2)
    for i, v := range fields[2:] {
        parsedInt, err := strconv.ParseUint(v, 10, 64)
        if err != nil {
            return fmt.Errorf("error parsing uint in tcp parser, idx: %d, actual: %v (%v)", i, v, err)
        }
        parsedInts[i] = parsedInt
    }

    // assign the parsed int values to the struct fields 
    t.Port = parsedInts[0]
    t.BindCount = parsedInts[1]
    t.ConnectCount = parsedInts[2]
    t.ConnectTime = parsedInts[3]
    t.IdleTime = parsedInts[4]
    t.RpcSends = parsedInts[5]
    t.RpcReceives = parsedInts[6]
    t.BadXids = parsedInts[7]
    t.InflightSends = parsedInts[8]
    t.BacklogUtil = parsedInts[9]

    // statvers 1.1+ extra fields conditionally added
    // im not sure if there's any situation where less 
    // than all three of these would be added, and thus 
    // not sure if checking the len() three times is 
    // needed, but we'll err on the side of preventing 
    // index out of bounds errors caused by arcane nfs 
    // kernel code because we do see the "1.1" fields 
    // being reported on mounts claiming to be 1.0 
    // so... something is....unknown.. yeah
    if len(parsedInts) >= 11 {
      t.MaxRPCSlots = parsedInts[10]
    }
    if len(parsedInts) >= 12 {
      t.CumSendingQueue = parsedInts[11]
    }
    if len(parsedInts) >= 13 {
      t.CumPendingQueue = parsedInts[12]
    }

    return nil
}


type NFSTransportCountersRDMA struct {
    Port             uint64
    BindCount        uint64
    ConnectCount     uint64
    ConnectTime      uint64
    IdleTime         uint64
    RpcSends         uint64
    RpcReceives      uint64
    BadXids          uint64
    BacklogUtil      uint64
    ReadChunks       uint64
    WriteChunks      uint64
    ReplyChunks      uint64
    TotalRdmaReq     uint64
    TotalRdmaRep     uint64
    Pullup           uint64
    Fixup            uint64
    Hardway          uint64
    FailedMarshal    uint64
    BadReply         uint64
}

func (r *NFSTransportCountersRDMA) Protocol() string {
  return "rdma"
}

func (r *NFSTransportCountersRDMA) ParseCounters(fields []string) error {
    if fields[0] != "xprt:" {
        return fmt.Errorf("malformed xprt: line in RDMA parser, expected 'xprt:', got %v", fields[0])
    }
    if fields[1] != "rdma" {
        return fmt.Errorf("malformed xprt: line in RDMA parser, expected 'rdma', got %v", fields[1])
    }
    if len(fields) < 21 {
        return fmt.Errorf("unexpected number of fields for RDMA, expected at least 21, got: %v", len(fields))
    }

    // parse the string fields after "xprt: rdma"
    parsedInts := make([]uint64, len(fields)-2)
    for i, v := range fields[2:] {
        parsedInt, err := strconv.ParseUint(v, 10, 64)
        if err != nil {
            return fmt.Errorf("error parsing uint in rdma parser, idx: %d, actual: %v (%v)", i, v, err)
        }
        parsedInts[i] = parsedInt
    }

    // assign the parsed values
    r.Port = parsedInts[0]
    r.BindCount = parsedInts[1]
    r.ConnectCount = parsedInts[2]
    r.ConnectTime = parsedInts[3]
    r.IdleTime = parsedInts[4]
    r.RpcSends = parsedInts[5]
    r.RpcReceives = parsedInts[6]
    r.BadXids = parsedInts[7]
    r.BacklogUtil = parsedInts[8]
    r.ReadChunks = parsedInts[9]
    r.WriteChunks = parsedInts[10]
    r.ReplyChunks = parsedInts[11]
    r.TotalRdmaReq = parsedInts[12]
    r.TotalRdmaRep = parsedInts[13]
    r.Pullup = parsedInts[14]
    r.Fixup = parsedInts[15]
    r.Hardway = parsedInts[16]
    r.FailedMarshal = parsedInts[17]
    r.BadReply = parsedInts[18]

    return nil
}

// ParsePerOpStats parses all of the additional data that begins with 
// `per-op statistics` in the mount device details section. It should be 
// a list of 8 int field counter values with a label.
// Returns an error if any of the parsing or string conversions fail.
func (i *NFSInfo) ParsePerOpStats(stats []string) error {
  if len(stats) <= 1 {
    return fmt.Errorf("expected many lines of RCP per op stats, got: %d", len(stats))
  }

  for _, line := range stats {
    // the lines from this section of the data have a lot of leading tabs and spaces
    // to make it pretty for humans to read. we need to trim them, and check/skip empty 
    // lines as well as the header line.
    line = strings.TrimSpace(line)
    if line == "" { continue }
    if line == "per-op statistics" { continue }

    // break our trimmed line up into fields, the first of which will be the op code
    // we'll make an 8 len array for the parsedInts. I _think_ it;s always 8 counter 
    // values but we'll use len() just incase
    fields := strings.Fields(line)
    intFields := make([]uint64, (len(fields)-1)) 

    // loop over all of the counter values and attempt to parse the int values into out slice 
    for idx, v := range fields {
      if idx == 0 { continue }
      converted, err := strconv.ParseInt(v, 10, 64)
      if err != nil {
        return fmt.Errorf("failed to parse int: %v, %v", v, err)
      }
      intFields[idx-1] = uint64(converted)
    }
    
    // trim the op code label, assign all of the parsed int values to a struct, and add this 
    // struct to the parent NFSInfo struct  
    op := strings.Trim(fields[0], ":")
    opstats := RPCOpStat{
      Operations: uint64(intFields[0]),
      Transmissions: uint64(intFields[1]),
      MajorTimeouts: uint64(intFields[2]),
      BytesSent: uint64(intFields[3]),
      BytesReceived: uint64(intFields[4]),
      CumQueueTime: uint64(intFields[5]) ,
      CumRespTime: uint64(intFields[6]) ,
      CumTotalReqTime: uint64(intFields[7]),
      ErrStats: uint64(0),
    }

    if len(intFields) >= 9 {
      opstats.ErrStats = uint64(intFields[8])
    }

    i.RPCOpStats[op] = opstats
  }
  return nil
}

// RPCOpStat holds the data for each line of "per-op" stats in an NFS mount. 
type RPCOpStat struct {
  Operations    uint64 
  Transmissions uint64 
  MajorTimeouts uint64 
  BytesSent     uint64 
  BytesReceived uint64
  CumQueueTime  uint64 
  CumRespTime   uint64 
  CumTotalReqTime uint64 
  ErrStats      uint64
}







