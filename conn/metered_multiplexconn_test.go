package meterconn

import (
	"bytes"
	"net"
	"time"

	tpt "github.com/libp2p/go-libp2p-transport"
	smux "github.com/libp2p/go-stream-muxer"
	ma "github.com/multiformats/go-multiaddr"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockMultiplexConn struct {
	streamToAccept *mockStream
	streamToOpen   *mockStream
}

var _ tpt.MultiplexConn = &mockMultiplexConn{}

func (c *mockMultiplexConn) AcceptStream() (smux.Stream, error) { return c.streamToAccept, nil }
func (c *mockMultiplexConn) OpenStream() (smux.Stream, error)   { return c.streamToOpen, nil }
func (c *mockMultiplexConn) Close() error                       { panic("not implemented") }
func (c *mockMultiplexConn) IsClosed() bool                     { panic("not implemented") }
func (c *mockMultiplexConn) LocalAddr() net.Addr                { panic("not implemented") }
func (c *mockMultiplexConn) LocalMultiaddr() ma.Multiaddr       { panic("not implemented") }
func (c *mockMultiplexConn) RemoteAddr() net.Addr               { panic("not implemented") }
func (c *mockMultiplexConn) RemoteMultiaddr() ma.Multiaddr      { panic("not implemented") }
func (c *mockMultiplexConn) Transport() tpt.Transport           { panic("not implemented") }

type mockStream struct {
	dataToRead  bytes.Buffer
	dataToWrite bytes.Buffer
}

var _ smux.Stream = &mockStream{}

func (s *mockStream) Read(b []byte) (int, error)       { return s.dataToRead.Read(b) }
func (s *mockStream) Write(b []byte) (int, error)      { return s.dataToWrite.Write(b) }
func (s *mockStream) Close() error                     { panic("not implemented") }
func (s *mockStream) Reset() error                     { panic("not implemented") }
func (s *mockStream) SetDeadline(time.Time) error      { panic("not implemented") }
func (s *mockStream) SetReadDeadline(time.Time) error  { panic("not implemented") }
func (s *mockStream) SetWriteDeadline(time.Time) error { panic("not implemented") }

var _ = Describe("MultiplexConn", func() {
	var (
		conn  *mockMultiplexConn
		mconn *MeteredMultiplexConn

		readCounter  *counter
		writeCounter *counter

		stream1 *mockStream
		stream2 *mockStream
	)

	BeforeEach(func() {
		readCounter = &counter{}
		writeCounter = &counter{}

		conn = &mockMultiplexConn{}
		mconn = newMeteredMultiplexConn(conn, readCounter.Count, writeCounter.Count)

		stream1 = &mockStream{}
		stream1.dataToRead.Write([]byte("foobar"))
		stream2 = &mockStream{}
		stream2.dataToRead.Write([]byte("foobar"))
	})

	It("counts data written on a single stream", func() {
		conn.streamToAccept = stream1

		str, err := mconn.AcceptStream()
		Expect(err).ToNot(HaveOccurred())
		n, _ := str.Read(make([]byte, 4))
		Expect(n).To(Equal(4))
		n, _ = str.Read(make([]byte, 1))
		Expect(n).To(Equal(1))
		Expect(readCounter.count).To(BeEquivalentTo(5))
		Expect(writeCounter.count).To(BeZero())
	})

	It("counts data read from a single stream", func() {
		conn.streamToAccept = &mockStream{}

		str, err := mconn.AcceptStream()
		Expect(err).ToNot(HaveOccurred())
		_, err = str.Write([]byte("foobar"))
		Expect(err).ToNot(HaveOccurred())
		_, err = str.Write([]byte("foo"))
		Expect(err).ToNot(HaveOccurred())
		Expect(readCounter.count).To(BeZero())
		Expect(writeCounter.count).To(BeEquivalentTo(9))
	})

	It("accumulates data written to multiple streams", func() {
		conn.streamToAccept = &mockStream{}
		conn.streamToOpen = &mockStream{}
		str1, err := mconn.AcceptStream()
		Expect(err).ToNot(HaveOccurred())
		_, err = str1.Write([]byte("foobar"))
		str2, err := mconn.OpenStream()
		Expect(err).ToNot(HaveOccurred())
		_, err = str2.Write([]byte("foo"))
		Expect(err).ToNot(HaveOccurred())
		Expect(readCounter.count).To(BeZero())
		Expect(writeCounter.count).To(BeEquivalentTo(9))
	})

	It("counts data read from multiple streams", func() {
		conn.streamToAccept = stream1
		conn.streamToOpen = stream2

		str1, err := mconn.AcceptStream()
		Expect(err).ToNot(HaveOccurred())
		p := make([]byte, 4)
		n, _ := str1.Read(p)
		Expect(n).To(Equal(4))
		str2, err := mconn.OpenStream()
		Expect(err).ToNot(HaveOccurred())
		p = make([]byte, 3)
		n, _ = str2.Read(p)
		Expect(n).To(Equal(3))
		Expect(readCounter.count).To(BeEquivalentTo(7))
		Expect(writeCounter.count).To(BeZero())
	})
})
