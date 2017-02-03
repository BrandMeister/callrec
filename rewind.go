package main

const rewindClassRewindControl = 0x0000
const rewindClassSystemConsole = 0x0100
const rewindClassApplication = 0x0900

const rewindPacketTypeKeepAlive = (rewindClassRewindControl + 0)
const rewindPacketTypeClose = (rewindClassRewindControl + 1)
const rewindPacketTypeChallenge = (rewindClassRewindControl + 2)
const rewindPacketTypeAuthentication = (rewindClassRewindControl + 3)

const rewindPacketTypeReport = (rewindClassSystemConsole + 0)

const rewindPacketTypeConfiguration = (rewindClassApplication + 0x00)
const rewindPacketTypeSubscription = (rewindClassApplication + 0x01)

const rewindPacketTypeDMRAudioFrame = (rewindClassApplication + 0x20)
const rewindPacketTypeDMRTerminatorWithLC = (rewindClassApplication + 0x12)
const rewindPacketTypeSuperHeader = (rewindClassApplication + 0x28)
const rewindPacketTypeFailureCode = (rewindClassApplication + 0x29)

const rewindProtocolSign = "REWIND01"
const rewindDataLength = (len(rewindProtocolSign) + 10)

type rewindData struct {
	Sign          [len(rewindProtocolSign)]byte
	PacketType    uint16 // rewindPacketType*
	Flags         uint16
	SeqNum        uint32
	PayloadLength uint16
	//Payload       []byte
}

const rewindRoleApplication = 0x20
const rewindServiceSimpleApplication = (rewindRoleApplication + 0)

const rewindVersionDescription = "Call recorder"
const rewindVersionDataLength = (len(rewindVersionDescription) + 5)

type rewindVersionData struct {
	RemoteID      uint32
	RewindService uint8                               // rewindService*
	Description   [len(rewindVersionDescription)]byte // Software name and version
}

const rewindOptionSuperHeader = (1 << 0)

type rewindConfigurationData struct {
	Options uint32 // rewindOption*
}

const rewindSessionTypePrivateVoice = 5
const rewindSessionTypeGroupVoice = 7

type rewindSubscriptionData struct {
	SessionType uint32 // rewindSessionType*
	DstID       uint32
}

const rewindCallLength = 10

type rewindSuperHeader struct {
	SessionType uint32 // rewindSessionType*
	SrcID       uint32
	DstID       uint32
	SrcCall     [rewindCallLength]byte
	DstCall     [rewindCallLength]byte
}
