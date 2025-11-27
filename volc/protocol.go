package volc

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/golang/glog"
)

var (
	errNoVersionAndSize              = errors.New("no protocol version and header size byte")
	errNoTypeAndFlag                 = errors.New("no message type and specific flag byte")
	errNoSerializationAndCompression = errors.New("no serialization and compression method byte")
	errRedundantBytes                = errors.New("there are redundant bytes in data")
	errInvalidMessageType            = errors.New("invalid message type bits")
	errInvalidSerialization          = errors.New("invalid serialization bits")
	errInvalidCompression            = errors.New("invalid compression bits")
	errNoEnoughHeaderBytes           = errors.New("no enough header bytes")
	errReadEvent                     = errors.New("read event number")
	errReadSessionIDSize             = errors.New("read session ID size")
	errReadConnectIDSize             = errors.New("read connection ID size")
	errReadPayloadSize               = errors.New("read payload size")
	errReadPayload                   = errors.New("read payload")
	errReadSequence                  = errors.New("read sequence number")
	errReadErrorCode                 = errors.New("read error code")
	errReadErrorSize                 = errors.New("read error size")
	errReadError                     = errors.New("read error")
)

type (
	// MsgType defines message type which determines how the message will be
	// serialized with the protocol.
	MsgType int32
	// MsgTypeFlagBits defines the 4-bit message-type specific flags. The specific
	// values should be defined in each specific usage scenario.
	MsgTypeFlagBits uint8

	// VersionBits defines the 4-bit version type.
	VersionBits uint8
	// HeaderSizeBits defines the 4-bit header-size type.
	HeaderSizeBits uint8
	// SerializationBits defines the 4-bit serialization method type.
	SerializationBits uint8
	// CompressionBits defines the 4-bit compression method type.
	CompressionBits uint8
)

// Values that a MsgType variable can take.
const (
	MsgTypeInvalid MsgType = iota
	MsgTypeFullClient
	MsgTypeAudioOnlyClient
	MsgTypeFullServer
	MsgTypeAudioOnlyServer
	MsgTypeFrontEndResultServer
	MsgTypeError

	MsgTypeServerACK = MsgTypeAudioOnlyServer
)

func (t MsgType) String() string {
	switch t {
	case MsgTypeFullClient:
		return "FullClient"
	case MsgTypeAudioOnlyClient:
		return "AudioOnlyClient"
	case MsgTypeFullServer:
		return "FullServer"
	case MsgTypeAudioOnlyServer:
		return "AudioOnlyServer/ServerACK"
	case MsgTypeError:
		return "Error"
	case MsgTypeFrontEndResultServer:
		return "TtsFrontEndResult"
	default:
		return fmt.Sprintf("invalid message type: %d", t)
	}
}

// Values that a MsgTypeFlagBits variable can take.
const (
	// For common protocol.
	MsgTypeFlagNoSeq       MsgTypeFlagBits = 0     // Non-terminal packet with no sequence
	MsgTypeFlagPositiveSeq MsgTypeFlagBits = 0b1   // Non-terminal packet with sequence > 0
	MsgTypeFlagLastNoSeq   MsgTypeFlagBits = 0b10  // last packet with no sequence
	MsgTypeFlagNegativeSeq MsgTypeFlagBits = 0b11  // last packet with sequence < 0
	MsgTypeFlagWithEvent   MsgTypeFlagBits = 0b100 // Payload contains event number (int32)
)

// Values that a VersionBits variable can take.
const (
	Version1 VersionBits = (iota + 1) << 4
	Version2
	Version3
	Version4
)

// Values that a HeaderSizeBits variable can take.
const (
	HeaderSize4 HeaderSizeBits = iota + 1
	HeaderSize8
	HeaderSize12
	HeaderSize16
)

// Values that a SerializationBits variable can take.
const (
	SerializationRaw    SerializationBits = 0
	SerializationJSON   SerializationBits = 0b1 << 4
	SerializationThrift SerializationBits = 0b11 << 4
	SerializationCustom SerializationBits = 0b1111 << 4
)

// Values that a CompressionBits variable can take.
const (
	CompressionNone   CompressionBits = 0
	CompressionGzip   CompressionBits = 0b1
	CompressionCustom CompressionBits = 0b1111
)

var (
	msgTypeToBits = map[MsgType]uint8{
		MsgTypeFullClient:           0b1 << 4,
		MsgTypeAudioOnlyClient:      0b10 << 4,
		MsgTypeFullServer:           0b1001 << 4,
		MsgTypeAudioOnlyServer:      0b1011 << 4,
		MsgTypeFrontEndResultServer: 0b1100 << 4,
		MsgTypeError:                0b1111 << 4,
	}
	bitsToMsgType = make(map[uint8]MsgType, len(msgTypeToBits))

	serializations = map[SerializationBits]bool{
		SerializationRaw:    true,
		SerializationJSON:   true,
		SerializationThrift: true,
		SerializationCustom: true,
	}

	compressions = map[CompressionBits]bool{
		CompressionNone:   true,
		CompressionGzip:   true,
		CompressionCustom: true,
	}
)

func init() {
	// Construct inverse mapping of msgTypeToBits.
	for msgType, bits := range msgTypeToBits {
		bitsToMsgType[bits] = msgType
	}
}

// ContainsSequenceFunc defines the functional type that checks whether the
// MsgTypeFlagBits indicates the existence of a sequence number in serialized
// data. The background is that not all responses contain a sequence number,
// and whether a response contains one depends on the the message type specific
// flag bits. What makes it more complicated is that this dependency varies in
// each use case (eg, TTS protocol has its own dependency specification, more
// details at: https://bytedance.feishu.cn/docs/doccn8MD4cZHQuvobbtouWfUVsV).
type ContainsSequenceFunc func(MsgTypeFlagBits) bool

// CompressFunc defines the functional type that does the compression operation.
type CompressFunc func([]byte) ([]byte, error)

type readFunc func(*bytes.Buffer) error
type writeFunc func(*bytes.Buffer) error

// Unmarshal deserializes the binary `data` into a Message and also returns
// the BinaryProtocol.
func Unmarshal(data []byte, containsSequence ContainsSequenceFunc) (*Message, *BinaryProtocol, error) {
	var (
		buf      = bytes.NewBuffer(data)
		readSize int
	)

	versionSize, err := buf.ReadByte()
	if err != nil {
		return nil, nil, errNoVersionAndSize
	}
	readSize++

	prot := &BinaryProtocol{
		versionAndHeaderSize: versionSize,
		containsSequence:     containsSequence,
	}
	glog.V(2).Infof("Read version: %04b", versionSize>>4)
	glog.V(2).Infof("Read size: %04b", versionSize&0b1111)

	typeAndFlag, err := buf.ReadByte()
	if err != nil {
		return nil, nil, errNoTypeAndFlag
	}
	readSize++
	glog.V(2).Infof("Read message type: %04b", typeAndFlag>>4)
	glog.V(2).Infof("Read message type specific flag: %04b", typeAndFlag&0b1111)

	msg, err := NewMessageFromByte(typeAndFlag)
	if err != nil {
		return nil, nil, err
	}

	serializationCompression, err := buf.ReadByte()
	if err != nil {
		return nil, nil, errNoSerializationAndCompression
	}
	glog.V(2).Infof("Read serialization method: %04b", serializationCompression>>4)
	glog.V(2).Infof("Read compression method: %04b", serializationCompression&0b1111)
	readSize++
	prot.serializationAndCompression = serializationCompression
	if _, ok := serializations[prot.Serialization()]; !ok {
		return nil, nil, fmt.Errorf("%w: %b", errInvalidSerialization, prot.Serialization())
	}
	if _, ok := compressions[prot.Compression()]; !ok {
		return nil, nil, fmt.Errorf("%w: %b", errInvalidCompression, prot.Compression())
	}

	// Read all the remaining zero-padding bytes in the header.
	if paddingSize := prot.HeaderSize() - readSize; paddingSize > 0 {
		if n, err := buf.Read(make([]byte, paddingSize)); err != nil || n < paddingSize {
			return nil, nil, fmt.Errorf("%w: %d", errNoEnoughHeaderBytes, n)
		}
	}

	readers, err := msg.readers(containsSequence)
	if err != nil {
		return nil, nil, err
	}
	for _, read := range readers {
		if err := read(buf); err != nil {
			return nil, nil, err
		}
	}

	if _, err := buf.ReadByte(); err != io.EOF {
		return nil, nil, errRedundantBytes
	}
	return msg, prot, nil
}

// Message defines the general message content type.
type Message struct {
	Type            MsgType
	typeAndFlagBits uint8

	Event     int32
	SessionID string
	ConnectID string
	Sequence  int32
	ErrorCode uint32
	// Raw payload (not Gzip compressed). BinaryProtocol.Marshal will do the
	// compression for you.
	Payload []byte
}

// NewMessage returns a new Message instance of the given message type with the
// specific flag.
func NewMessage(msgType MsgType, typeFlag MsgTypeFlagBits) (*Message, error) {
	bits, ok := msgTypeToBits[msgType]
	if !ok {
		return nil, fmt.Errorf("invalid message type: %d", msgType)
	}
	return &Message{
		Type:            msgType,
		typeAndFlagBits: bits + uint8(typeFlag),
	}, nil
}

// NewMessageFromByte reads the byte as the message type and specific flag bits
// and composes a new Message instance from them.
func NewMessageFromByte(typeAndFlag byte) (*Message, error) {
	bits := typeAndFlag &^ 0b00001111
	msgType, ok := bitsToMsgType[bits]
	if !ok {
		return nil, fmt.Errorf("%w: %b", errInvalidMessageType, bits>>4)
	}
	return &Message{
		Type:            msgType,
		typeAndFlagBits: typeAndFlag,
	}, nil
}

// TypeFlag returns the message type specific flag.
func (m *Message) TypeFlag() MsgTypeFlagBits {
	return MsgTypeFlagBits(m.typeAndFlagBits &^ 0b11110000)
}

func (m *Message) writers(compress CompressFunc) (writers []writeFunc, _ error) {
	if compress != nil {
		payload, err := compress(m.Payload)
		if err != nil {
			return nil, fmt.Errorf("compress payload failed: %w", err)
		}
		m.Payload = payload
	}

	if containsSequence(m.TypeFlag()) {
		writers = append(writers, m.writeSequence)
		glog.Info("Add Sequence writer.")
	}

	if containsEvent(m.TypeFlag()) {
		writers = append(writers, m.writeEvent, m.writeSessionID)
		glog.V(1).Info("Add Event and SessionID writer.")
	}

	writers = append(writers, m.writePayload)
	glog.V(1).Info("Add Payload writers.")
	return writers, nil
}

func (m *Message) writeEvent(buf *bytes.Buffer) error {
	if err := binary.Write(buf, binary.BigEndian, m.Event); err != nil {
		return fmt.Errorf("write sequence number (%d): %w", m.Event, err)
	}
	return nil
}

func (m *Message) writeSessionID(buf *bytes.Buffer) error {
	switch m.Event {
	case 1, 2, 50, 51, 52: // StartConnection, FinishConnection, ConnectionStarted, ConnectionFailed, ConnectionFinished
		glog.V(1).Infof("Skip writing session ID for event: %d", m.Event)
		return nil
	}

	size := len(m.SessionID)
	if size > math.MaxUint32 {
		return fmt.Errorf("payload size (%d) exceeds max(uint32)", size)
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(size)); err != nil {
		return fmt.Errorf("write payload size (%d): %w", size, err)
	}
	buf.WriteString(m.SessionID)
	return nil
}

func (m *Message) writeSequence(buf *bytes.Buffer) error {
	if err := binary.Write(buf, binary.BigEndian, m.Sequence); err != nil {
		return fmt.Errorf("write sequence number (%d): %w", m.Sequence, err)
	}
	return nil
}

func (m *Message) writeErrorCode(buf *bytes.Buffer) error {
	if err := binary.Write(buf, binary.BigEndian, m.ErrorCode); err != nil {
		return fmt.Errorf("write error code (%d): %w", m.ErrorCode, err)
	}
	return nil
}

func (m *Message) writePayload(buf *bytes.Buffer) error {
	size := len(m.Payload)
	if size > math.MaxUint32 {
		return fmt.Errorf("payload size (%d) exceeds max(uint32)", size)
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(size)); err != nil {
		return fmt.Errorf("write payload size (%d): %w", size, err)
	}
	buf.Write(m.Payload)
	return nil
}

func (m *Message) readers(containsSequence ContainsSequenceFunc) (readers []readFunc, _ error) {

	switch m.Type {
	case MsgTypeFullClient, MsgTypeFullServer, MsgTypeFrontEndResultServer:

	case MsgTypeAudioOnlyClient:
		if containsSequence == nil || containsSequence(m.TypeFlag()) {
			readers = append(readers, m.readSequence)
			glog.V(1).Info("AudioOnlyClient message: add Sequence reader.")
		}

	case MsgTypeAudioOnlyServer:
		if containsSequence != nil && containsSequence(m.TypeFlag()) {
			readers = append(readers, m.readSequence)
			glog.V(1).Info("AudioOnlyServer message: add Sequence reader.")
		}

	case MsgTypeError:
		readers = append(readers, m.readErrorCode)
		glog.V(1).Info("Error message: add Error-Code reader.")

	default:
		return nil, fmt.Errorf("cannot deserialize message with invalid type: %d", m.Type)
	}

	if containsEvent(m.TypeFlag()) {
		readers = append(readers, m.readEvent, m.readSessionID, m.readConnectID)
		glog.V(1).Info("Add Event and SessionID readers.")
	}

	readers = append(readers, m.readPayload)
	glog.V(1).Info("Add Payload reader.")
	return readers, nil
}

func (m *Message) readEvent(buf *bytes.Buffer) error {
	if err := binary.Read(buf, binary.BigEndian, &m.Event); err != nil {
		return fmt.Errorf("%w: %v", errReadEvent, err)
	}
	glog.V(1).Infof("Read Event: %d", m.Event)
	return nil
}

func (m *Message) readSessionID(buf *bytes.Buffer) error {
	switch m.Event {
	case 1, 2, 50, 51, 52: //StartConnection, FinishConnection, ConnectionStarted, ConnectionFailed, ConnectionFinished
		glog.V(1).Infof("Skip reading session ID for event: %d", m.Event)
		return nil
	}

	var size uint32
	if err := binary.Read(buf, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("%w: %v", errReadSessionIDSize, err)
	}
	glog.V(2).Infof("Read SessionID length: %d", size)

	if size > 0 {
		m.SessionID = string(buf.Next(int(size)))
	}
	glog.V(2).Infof("Read SessionID content: %s", m.SessionID)
	return nil
}

func (m *Message) readConnectID(buf *bytes.Buffer) error {
	switch m.Event {
	case 50, 51, 52: // ConnectionStarted, event.Type_ConnectionFailed, ConnectionFinished
	default:
		glog.V(1).Infof("Skip reading session ID for event: %d", m.Event)
		return nil
	}

	var size uint32
	if err := binary.Read(buf, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("%w: %v", errReadConnectIDSize, err)
	}
	glog.V(2).Infof("Read connection ID length: %d", size)

	if size > 0 {
		m.ConnectID = string(buf.Next(int(size)))
	}
	glog.V(2).Infof("Read connection ID content: %s", m.ConnectID)
	return nil
}

func (m *Message) readSequence(buf *bytes.Buffer) error {
	if err := binary.Read(buf, binary.BigEndian, &m.Sequence); err != nil {
		return fmt.Errorf("%w: %v", errReadSequence, err)
	}
	glog.V(2).Infof("Read Sequence: %d", m.Sequence)
	return nil
}

func (m *Message) readErrorCode(buf *bytes.Buffer) error {
	if err := binary.Read(buf, binary.BigEndian, &m.ErrorCode); err != nil {
		return fmt.Errorf("%w: %v", errReadErrorCode, err)
	}
	glog.V(2).Infof("Read ErrorCode: %d", m.ErrorCode)
	return nil
}

func (m *Message) readPayload(buf *bytes.Buffer) error {
	var size uint32
	if err := binary.Read(buf, binary.BigEndian, &size); err != nil {
		return fmt.Errorf("%w: %v", errReadPayloadSize, err)
	}
	glog.V(2).Infof("Read Payload length: %d", size)

	if size > 0 {
		m.Payload = buf.Next(int(size))
	}
	if m.Type == MsgTypeFullClient || m.Type == MsgTypeFullServer || m.Type == MsgTypeError {
		glog.V(2).Infof("Read Payload content: %s", m.Payload)
	}
	return nil
}

// ContainsSequence reports whether a message type specific flag indicates
// messages with this kind of flag contain a sequence number in its serialized
// value. This determiner function should be used for common binary protocol.
func ContainsSequence(bits MsgTypeFlagBits) bool {
	return bits&MsgTypeFlagPositiveSeq == MsgTypeFlagPositiveSeq || bits&MsgTypeFlagNegativeSeq == MsgTypeFlagNegativeSeq
}

func containsEvent(bits MsgTypeFlagBits) bool {
	return bits&MsgTypeFlagWithEvent == MsgTypeFlagWithEvent
}

func containsSequence(bits MsgTypeFlagBits) bool {
	return bits&MsgTypeFlagPositiveSeq == MsgTypeFlagPositiveSeq || bits&MsgTypeFlagNegativeSeq == MsgTypeFlagNegativeSeq
}

// BinaryProtocol implements the binary protocol serialization and deserialization
// used in Lab-Speech MDD, TTS, ASR, etc. services. For more details, read:
// https://bytedance.feishu.cn/docs/doccnT0t71J4LCQCS0cnB4Eca8D
type BinaryProtocol struct {
	versionAndHeaderSize        uint8
	serializationAndCompression uint8

	containsSequence ContainsSequenceFunc
	compress         CompressFunc
}

// NewBinaryProtocol returns a new BinaryProtocol instance.
func NewBinaryProtocol() *BinaryProtocol {
	return new(BinaryProtocol)
}

// Clone returns a clone of current BinaryProtocol
func (p *BinaryProtocol) Clone() *BinaryProtocol {
	clonedBinaryProtocal := new(BinaryProtocol)
	clonedBinaryProtocal.versionAndHeaderSize = p.versionAndHeaderSize
	clonedBinaryProtocal.serializationAndCompression = p.serializationAndCompression
	clonedBinaryProtocal.containsSequence = p.containsSequence
	clonedBinaryProtocal.compress = p.compress
	return clonedBinaryProtocal
}

// SetVersion sets the protocol version.
func (p *BinaryProtocol) SetVersion(v VersionBits) {
	// Clear the higher 4 bits in `p.versionAndHeaderSize` and reset them to `v`.
	p.versionAndHeaderSize = (p.versionAndHeaderSize &^ 0b11110000) + uint8(v)
}

// Version returns the integral version value.
func (p *BinaryProtocol) Version() int {
	return int(p.versionAndHeaderSize >> 4)
}

// SetHeaderSize sets the protocol header size.
func (p *BinaryProtocol) SetHeaderSize(s HeaderSizeBits) {
	// Clear the lower 4 bits in `p.versionAndHeaderSize` and reset them to `s`.
	p.versionAndHeaderSize = (p.versionAndHeaderSize &^ 0b00001111) + uint8(s)
}

// HeaderSize returns the protocol header size.
func (p *BinaryProtocol) HeaderSize() int {
	return 4 * int(p.versionAndHeaderSize&^0b11110000)
}

// SetSerialization sets the serialization method.
func (p *BinaryProtocol) SetSerialization(s SerializationBits) {
	// Clear the higher 4 bits in `p.serializationAndCompression` and reset them to `s`.
	p.serializationAndCompression = (p.serializationAndCompression &^ 0b11110000) + uint8(s)
}

// Serialization returns the bits value of protocol serialization method.
func (p *BinaryProtocol) Serialization() SerializationBits {
	return SerializationBits(p.serializationAndCompression &^ 0b00001111)
}

// SetCompression sets the compression method.
func (p *BinaryProtocol) SetCompression(c CompressionBits, f CompressFunc) {
	// Clear the lower 4 bits in `p.serializationAndCompression` and reset them to `c`.
	p.serializationAndCompression = (p.serializationAndCompression &^ 0b00001111) + uint8(c)
	p.compress = f
}

// Compression returns the bits value of protocol compression method.
func (p *BinaryProtocol) Compression() CompressionBits {
	return CompressionBits(p.serializationAndCompression &^ 0b11110000)
}

// Marshal serializes the message to a sequence of binary data.
func (p *BinaryProtocol) Marshal(msg *Message) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := p.writeHeader(buf, msg); err != nil {
		return nil, fmt.Errorf("write header: %w", err)
	}

	writers, err := msg.writers(p.compress)
	if err != nil {
		return nil, err
	}
	for _, write := range writers {
		if err := write(buf); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (p *BinaryProtocol) writeHeader(buf *bytes.Buffer, msg *Message) error {
	return binary.Write(buf, binary.BigEndian, p.header(msg))
}

func (p *BinaryProtocol) header(msg *Message) []byte {
	header := []uint8{
		p.versionAndHeaderSize,
		msg.typeAndFlagBits,
		p.serializationAndCompression,
	}
	if padding := p.HeaderSize() - len(header); padding > 0 {
		header = append(header, make([]uint8, padding)...)
	}
	return header
}
