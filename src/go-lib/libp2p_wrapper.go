package main

// #include "libp2p_definitions.h"
import "C"

import (
	"log"
	"fmt"
	"math/rand"
	"unsafe"
	"context"
	"time"
	"net"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/libp2p/go-libp2p/core/network"

	"github.com/libp2p/go-libp2p/core/peer"
	//"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	//"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	"github.com/multiformats/go-multiaddr"

	logging "github.com/ipfs/go-log"
	manet "github.com/multiformats/go-multiaddr/net"

	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	"sync"
	"reflect"
	"bufio"
)


// safeConvertToInt converts the C.size_t to an int, and returns a boolean
// indicating if the conversion was lossless and semantically equivalent.
func safeConvertToInt(n C.size_t) (int, bool) {
	return int(n), C.size_t(int(n)) == n && int(n) >= 0
}


type randomReader struct {
	random *rand.Rand
}

func (reader randomReader) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i++ {
		p[i] = byte(reader.random.Intn(256))
	}
	return len(p), nil
}

// Deterministic key.
func getIdentity(seed int64) (peer.ID, libp2p.Option) {
	reader := randomReader{
		random: rand.New(rand.NewSource(seed)),
	}
	privKey, _, err := crypto.GenerateEd25519Key(&reader)
	if err != nil {
		panic(err)
	}
	peerID, err := peer.IDFromPrivateKey(privKey)
	if err != nil {
		panic(err)
	}
	identity := libp2p.Identity(privKey)
	return peerID, identity
}

func getClientHostOptions(identity libp2p.Option, listenAddr string, serverAddrInfo peer.AddrInfo) []libp2p.Option {
	return []libp2p.Option{
		identity,
		libp2p.AddrsFactory(func(m []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return multiaddr.FilterAddrs(m, manet.IsPublicAddr)
		}),
		//libp2p.ForceReachabilityPrivate(),
		libp2p.ListenAddrStrings(listenAddr),
		//libp2p.EnableAutoRelay(),
		libp2p.EnableAutoRelayWithStaticRelays([]peer.AddrInfo{serverAddrInfo}),
		libp2p.EnableHolePunching(),
	}
}

type GoLibp2pHost struct {
	host *host.Host
}

type GoLibp2pStream struct {
	stream *network.Stream
}

//export makeBasicHost
func makeBasicHost(seed C.int64_t, listenPort C.int64_t) (C.Libp2pHostResult) {
	_, hostIdentity := getIdentity(int64(seed))

	opts := []libp2p.Option{
		hostIdentity,
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", int64(listenPort))),
		libp2p.DisableRelay(),
	}

	host, err := libp2p.New(opts...)
	if err != nil {
		log.Fatalf("Error creating host: %v\n", err)
		return C.Libp2pHostResult{
			error: mallocError(err),
		}
	}

	log.Printf("Create host success, listen port %d\n",  int64(listenPort));
	log.Printf("Host ID: %v\n", host.ID())

	//defer host.Close()*/
	return C.Libp2pHostResult{
		host: (*C.Libp2pHost)(mallocHandle(universe.Add(&GoLibp2pHost{&host}))),
	}
}

//export CreateHost
func CreateHost(listenAddr *C.libp2p_const_char, seed C.int64_t, relayAddr *C.libp2p_const_char, relayID *C.libp2p_const_char) (C.Libp2pHostResult) {
	//serverID, _ := getIdentity(0)
	logging.SetLogLevel("p2p-holepunch", "DEBUG")
	logging.SetLogLevel("relay", "DEBUG")
	identify.ActivationThresh = 1

	_, hostIdentity := getIdentity(int64(seed))

	//identity = dialerIdentity

	var serverID peer.ID

	if relayID != nil {
		serverID, _ = peer.Decode(C.GoString(relayID)) //TODO check decode
	} else {
		serverID, _ = getIdentity(0)
	}

	addr, err := multiaddr.NewMultiaddr(C.GoString(relayAddr))
	if err != nil {
		return C.Libp2pHostResult{
			error: mallocError(err),
		}
	}

	serverAddrInfo := peer.AddrInfo{
		ID:    serverID,
		Addrs: []multiaddr.Multiaddr{addr},
	}
	log.Printf("Server AddrInfo: %v\n", serverAddrInfo)

	hostOptions := getClientHostOptions(hostIdentity, C.GoString(listenAddr), serverAddrInfo)
	host, err := libp2p.New(hostOptions...)
	if err != nil {
		log.Fatalf("Error creating host: %v\n", err)
		return C.Libp2pHostResult{
			error: mallocError(err),
		}
	}

	log.Printf("Create host success")
	log.Printf("Host ID: %v\n", host.ID())

	//defer host.Close()*/
	return C.Libp2pHostResult{
		host: (*C.Libp2pHost)(mallocHandle(universe.Add(&GoLibp2pHost{&host}))),
	}
}

//export Connect
func Connect(hostHandle *C.Libp2pHost, addr *C.libp2p_const_char, peerID *C.libp2p_const_char) (*C.Libp2pError){

	h, ok := universe.Get(hostHandle._handle).(*GoLibp2pHost)

	if !ok {
		return mallocError(ErrInvalidHandle.New("host"))
	}

	derefHost := *h
	host := *(derefHost.host)

	var serverID peer.ID

	if peerID != nil {
		serverID, _ = peer.Decode(C.GoString(peerID)) //TODO check ID
	} else {
		serverID, _ = getIdentity(0)
	}
	log.Printf("serverID: %s - %s\n", serverID, C.GoString(addr))

	//relayaddr, err := multiaddr.NewMultiaddr("/p2p/" + serverID.String() + "/p2p-circuit/p2p/" + listenerID.String())

	address, err := multiaddr.NewMultiaddr(C.GoString(addr))
	log.Print(address)
	if err != nil {
		log.Println(err)
		return mallocError(err)
	}

	serverAddrInfo := peer.AddrInfo{
		ID:    serverID,
		Addrs: []multiaddr.Multiaddr{address},
	}
	log.Printf("Server AddrInfo: %v\n", serverAddrInfo)

	ctx := context.Background()

	for {
		log.Printf("Connecting to server %v...\n", serverAddrInfo)
		err := host.Connect(ctx, serverAddrInfo)
		if err != nil {
			log.Printf("Error connecting to server: %v\n", err)
		} else {
			log.Println("Connected to server.")
			break
		}
		time.Sleep(1 * time.Second)
	}

	return nil

}

//export WaitForPublicAddress
func WaitForPublicAddress(hostHandle *C.Libp2pHost, attempts uint32, delay uint32) (*C.Libp2pError) {

	h, ok := universe.Get(hostHandle._handle).(*GoLibp2pHost)

	if !ok {
		return mallocError(ErrInvalidHandle.New("host"))
	}

	derefHost := *h
	host := *(derefHost.host)

	// Wait until external addresses is observed with server's NAT service.
	idService := host.(*autorelay.AutoRelayHost).Host.(*basichost.BasicHost).IDService()
	for i := uint32(0); i < attempts; i++ {
		log.Printf("Attempt: %d\n", i + 1)
		for _, addr := range idService.OwnObservedAddrs() {
			if manet.IsPublicAddr(addr) {
				log.Printf("Observed self Addrs: %v\n", idService.OwnObservedAddrs())
				return nil
			}
		}
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	return mallocErrorRaw(0, "Can't get public address")
}

//export OpenStream
func OpenStream(hostHandle *C.Libp2pHost, streamName *C.libp2p_const_char, peerID *C.libp2p_const_char) (C.Libp2pOpenStreamResult){
	h, ok := universe.Get(hostHandle._handle).(*GoLibp2pHost)

	if !ok {
		return C.Libp2pOpenStreamResult {
			error: mallocError(ErrInvalidHandle.New("host")),
		}
	}

	derefHost := *h
	host := *(derefHost.host)

	ctx := context.Background()

	peer, _ := peer.Decode(C.GoString(peerID)) //TODO check decode

	stream, err := host.NewStream(ctx, peer, protocol.ID(C.GoString(streamName)))
	if err != nil {
		log.Println(err)
		return C.Libp2pOpenStreamResult {
			error: mallocError(err),
		}
	}

	log.Println("OpenStream Success");
	return C.Libp2pOpenStreamResult{
		stream: (*C.Libp2pStream)(mallocHandle(universe.Add(&GoLibp2pStream{&stream}))),
	}
}

//export WriteToStream
func WriteToStream(streamHandle *C.Libp2pStream, data *C.libp2p_const_char) (*C.Libp2pError) {
	s, ok := universe.Get(streamHandle._handle).(*GoLibp2pStream)

	if !ok {
		return mallocError(ErrInvalidHandle.New("stream"))
	}

	derefStream := *s
	stream := *(derefStream.stream)

	_, err := stream.Write([]byte(C.GoString(data)))
	//_, err := stream.Write([]byte("Hello, world!\n"))
	if err != nil {
		log.Fatalln(err)
		return mallocError(err)
	}
	return nil
}


//export ReadFromStream
func ReadFromStream(streamHandle *C.Libp2pStream, bytes unsafe.Pointer, length C.size_t) (C.Libp2pReadResult) {
	log.Println("ReadFromStream")
	s, ok := universe.Get(streamHandle._handle).(*GoLibp2pStream)

	if !ok {
		return C.Libp2pReadResult {
			error: mallocError(ErrInvalidHandle.New("stream")),
		}
	}

	derefStream := *s
	stream := *(derefStream.stream)

	ilength, ok := safeConvertToInt(length)
	if !ok {
		return C.Libp2pReadResult {
			error: mallocError(ErrInvalidArg.New("length too large")),
		}
	}

	var buf []byte
	hbuf := (*reflect.SliceHeader)(unsafe.Pointer(&buf))
	hbuf.Data = uintptr(bytes)
	hbuf.Len = ilength
	hbuf.Cap = ilength

	bufReader := bufio.NewReaderSize(stream, ilength)
	log.Println("Before read")
	n, err := bufReader.Read(buf)
	log.Println("After read")

	return C.Libp2pReadResult{
		bytes_read: C.size_t(n),
		error:      mallocError(err),
	}

}

//export StreamClose
func StreamClose(streamHandle *C.Libp2pStream) (*C.Libp2pError) {
	s, ok := universe.Get(streamHandle._handle).(*GoLibp2pStream)

	if !ok {
		return mallocError(ErrInvalidHandle.New("stream"))
	}

	derefStream := *s
	stream := *(derefStream.stream)
	stream.Close()
	return nil
}

//export ListenStreamBlock
func ListenStreamBlock(hostHandle *C.Libp2pHost, streamName *C.libp2p_const_char) (C.Libp2pOpenStreamResult) {

	h, ok := universe.Get(hostHandle._handle).(*GoLibp2pHost)

	if !ok {
		return C.Libp2pOpenStreamResult {
			error: mallocError(ErrInvalidHandle.New("host")),
		}
	}

	derefHost := *h
	host := *(derefHost.host)

	streamCh := make(chan network.Stream)

	// Use a mutex to protect access to the channel
	var mu sync.Mutex

	// Set up a stream handler
	host.SetStreamHandler(protocol.ID(C.GoString(streamName)), func(stream network.Stream) {
		// Lock before sending on the channel
		mu.Lock()
		streamCh <- stream
		mu.Unlock()
	})

	log.Println("Listening for incoming streams...")

	// Wait for a new stream
	stream := <-streamCh
	log.Println("Received a new stream!")

	// Handle the incoming stream here

	// Keep the main goroutine alive
	//select {}
	return C.Libp2pOpenStreamResult{
		stream: (*C.Libp2pStream)(mallocHandle(universe.Add(&GoLibp2pStream{&stream}))),
	}
}

//export ListenStream
func ListenStream(hostHandle *C.Libp2pHost, streamName *C.libp2p_const_char, onStream C.OnStreamCallback, additionalParams C.voidPtr) (*C.Libp2pError){
	h, ok := universe.Get(hostHandle._handle).(*GoLibp2pHost)

	if !ok {
		return mallocError(ErrInvalidHandle.New("host"))
	}

	derefHost := *h
	host := *(derefHost.host)

	// Set up a stream handler
	host.SetStreamHandler(protocol.ID(C.GoString(streamName)), func(stream network.Stream) {
		log.Println("---Received a new stream!---")
		C.NewStream(onStream, additionalParams, (*C.Libp2pStream)(mallocHandle(universe.Add(&GoLibp2pStream{&stream}))))
	})
	log.Println("---Register callback---")
	return nil
}

//export UnsafeFunc
func UnsafeFunc(iFunctionPointer C.Callback) {
	C.bridge_callback(iFunctionPointer, C.CString("Test string"), 5)
}


func handleConnection(conn net.Conn, stream network.Stream) {
	defer conn.Close()
	defer stream.Close()

	// Create channels for bidirectional communication
	done := make(chan struct{})

	// Read from the Unix socket and write to the libp2p stream
	go func() {
		buffer := make([]byte, 1024)
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				fmt.Println("Error reading from Unix socket:", err)
				close(done)
				return
			}

			// Assuming you have a libp2p stream, write the data to it
			_, err = stream.Write(buffer[:n])
			if err != nil {
				fmt.Println("Error writing to libp2p stream:", err)
				close(done)
				return
			}
		}
	}()

	// Read from the libp2p stream and write to the Unix socket
	buffer := make([]byte, 1024)
	for {
		n, err := stream.Read(buffer)
		if err != nil {
			fmt.Println("Error reading from libp2p stream:", err)
			close(done)
			return
		}

		// Write the data to the Unix socket
		_, err = conn.Write(buffer[:n])
		if err != nil {
			fmt.Println("Error writing to Unix socket:", err)
			close(done)
			return
		}
	}

	// Wait for either side to close the connection
	<-done
}

//export ConnectUnixSocketWithLibp2pStream
func ConnectUnixSocketWithLibp2pStream(streamHandle *C.Libp2pStream,  unixSocketFd C.int32_t)  (*C.Libp2pError){
	file := os.NewFile(uintptr(unixSocketFd), "")
	conn, err := net.FileConn(file)
	if err != nil {
		fmt.Println("Error creating net.Conn from file descriptor:", err)
		return mallocError(err)
	}

	s, ok := universe.Get(streamHandle._handle).(*GoLibp2pStream)

	if !ok {
		return mallocError(ErrInvalidHandle.New("stream"))
	}

	derefStream := *s
	stream := *(derefStream.stream)
	go handleConnection(conn, stream)
	return nil
}

func main() {
	// We need the main function to make possible
	// CGO compiler to compile the package as C shared library
}

var universe = newHandles()