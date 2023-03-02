package dataverification

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"github.com/filefilego/filefilego/internal/common"
	"github.com/filefilego/filefilego/internal/common/hexutil"
	"github.com/filefilego/filefilego/internal/contract"
	ffgcrypto "github.com/filefilego/filefilego/internal/crypto"
	"github.com/filefilego/filefilego/internal/node/protocols/messages"
	"github.com/filefilego/filefilego/internal/storage"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	// ReceiveMerkleTreeProtocolID is a protocol which receives the merkle tree nodes.
	ReceiveMerkleTreeProtocolID = "/ffg/dataverification_receive_merkletree/1.0.0"

	// FileTransferProtocolID is a protocol which is used to transfer files from file hoster to downloader node.
	FileTransferProtocolID = "/ffg/dataverification_file_transfer/1.0.0"

	// ReceiveKeyIVRandomizedFileSegmentsAndDataProtocolID is a protocol which receives the encryotion data and the raw unencrypted file segments to verifier.
	ReceiveKeyIVRandomizedFileSegmentsAndDataProtocolID = "/ffg/dataverification_receive_keyivrandomsegments_data/1.0.0"

	// EncryptionDataTransferProtocolID is a protocol which transfers the key data from verifier to file requester.
	EncryptionDataTransferProtocolID = "/ffg/dataverification_encryption_data_transfer/1.0.0"

	// ContractTransferProtocolID is a protocol which transfers download contracts between nodes.
	ContractTransferProtocolID = "/ffg/dataverification_contract_transfer/1.0.0"

	// ContractVerifierAcceptanceProtocolID is a protocol which accepts incoming download contracts and seal them by verifier.
	ContractVerifierAcceptanceProtocolID = "/ffg/dataverification_contract_accept/1.0.0"

	// deadlineTimeInSecond = 10

	bufferSize = 8192

	verifierSubDirectory = "verifications"
)

// Interface specifies the data verification functionalities.
type Interface interface {
	SendContractForAcceptance(ctx context.Context, verifierID peer.ID, request *messages.DownloadContractProto) error
	TransferContract(ctx context.Context, peerID peer.ID, request *messages.DownloadContractProto) error
	DecryptFile(filePath, decryptedFilePath string, key, iv []byte, encryptionType common.EncryptionType, randomizedFileSegments []int) (string, error)
	RequestEncryptionData(ctx context.Context, verifierID peer.ID, request *messages.KeyIVRequestProto) (*messages.KeyIVRandomizedFileSegmentsEnvelopeProto, error)
	SendFileMerkleTreeNodesToVerifier(ctx context.Context, verifierID peer.ID, request *messages.MerkleTreeNodesOfFileContractProto) error
	SendKeyIVRandomizedFileSegmentsAndDataToVerifier(ctx context.Context, verifierID peer.ID, filePath string, contractHash string, fileHash []byte) error
	RequestFileTransfer(ctx context.Context, fileHosterID peer.ID, request *messages.FileTransferInfoProto) (string, error)
}

// Protocol wraps the data verification protocols and handlers
type Protocol struct {
	host                    host.Host
	contractStore           contract.Interface
	storage                 storage.Interface
	merkleTreeTotalSegments int
	encryptionPercentage    int
	downloadDirectory       string
	dataVerifier            bool
}

// New creates a data verification protocol.
func New(h host.Host, contractStore contract.Interface, storage storage.Interface, merkleTreeTotalSegments, encryptionPercentage int, downloadDirectory string, dataVerifier bool) (*Protocol, error) {
	if h == nil {
		return nil, errors.New("host is nil")
	}

	if contractStore == nil {
		return nil, errors.New("contract store is nil")
	}

	if storage == nil {
		return nil, errors.New("storage is nil")
	}

	if downloadDirectory == "" {
		return nil, errors.New("download directory is empty")
	}

	p := &Protocol{
		host:                    h,
		contractStore:           contractStore,
		storage:                 storage,
		merkleTreeTotalSegments: merkleTreeTotalSegments,
		encryptionPercentage:    encryptionPercentage,
		downloadDirectory:       downloadDirectory,
		dataVerifier:            dataVerifier,
	}

	// the following protocols are hanlded by verifier
	if p.dataVerifier {
		p.host.SetStreamHandler(ReceiveMerkleTreeProtocolID, p.handleIncomingMerkleTreeNodes)
		p.host.SetStreamHandler(ContractVerifierAcceptanceProtocolID, p.handleIncomingContractVerifierAcceptance)
		p.host.SetStreamHandler(ReceiveKeyIVRandomizedFileSegmentsAndDataProtocolID, p.handleIncomingKeyIVRandomizedFileSegmentsAndData)
		p.host.SetStreamHandler(EncryptionDataTransferProtocolID, p.handleIncomingEncryptionDataTransfer)
	}

	p.host.SetStreamHandler(FileTransferProtocolID, p.handleIncomingFileTransfer)
	p.host.SetStreamHandler(ContractTransferProtocolID, p.handleIncomingContract)

	return p, nil
}

// handleIncomingContractVerifierAcceptance handles incoming contracts to verifier nodes for acceptance.
// verifier signs the contract and sends it back.
func (d *Protocol) handleIncomingContractVerifierAcceptance(s network.Stream) {
	// TODO
}

// TransferContract transfers a contract to a node.
func (d *Protocol) SendContractForAcceptance(ctx context.Context, verifierID peer.ID, request *messages.DownloadContractProto) error {
	// TODO
	return nil
}

// handleIncomingContract handles incoming contracts from nodes.
func (d *Protocol) handleIncomingContract(s network.Stream) {
	// TODO
}

// TransferContract transfers a contract to a node.
func (d *Protocol) TransferContract(ctx context.Context, peerID peer.ID, request *messages.DownloadContractProto) error {
	// TODO
	return nil
}

// DecryptFile descrypts a file given the file's encryption setup.
func (d *Protocol) DecryptFile(filePath, decryptedFilePath string, key, iv []byte, encryptionType common.EncryptionType, randomizedFileSegments []int) (string, error) {
	inputFile, err := os.OpenFile(filePath, os.O_RDWR, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to open input file in decryptFile: %w", err)
	}
	defer inputFile.Close()

	inputStats, err := inputFile.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get stats of input file in decryptFile: %w", err)
	}

	encryptor, err := common.NewEncryptor(encryptionType, key, iv)
	if err != nil {
		return "", fmt.Errorf("failed to create a new encryptor in decryptFile: %w", err)
	}

	outputFile, err := os.OpenFile(decryptedFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to open output file in decryptFile: %w", err)
	}

	err = common.DecryptFileSegments(int(inputStats.Size()), d.merkleTreeTotalSegments, d.encryptionPercentage, randomizedFileSegments, inputFile, outputFile, encryptor)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt file segments in decryptFile: %w", err)
	}
	return decryptedFilePath, nil
}

// RequestEncryptionData requests the encryption data from a verifier.
func (d *Protocol) RequestEncryptionData(ctx context.Context, verifierID peer.ID, request *messages.KeyIVRequestProto) (*messages.KeyIVRandomizedFileSegmentsEnvelopeProto, error) {
	s, err := d.host.NewStream(ctx, verifierID, EncryptionDataTransferProtocolID)
	if err != nil {
		return nil, fmt.Errorf("failed to create new stream to verifier for getting encryption data: %w", err)
	}
	defer s.Close()

	// future := time.Now().Add(deadlineTimeInSecond * time.Second)
	// err = s.SetDeadline(future)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to set encryption data for verifier stream deadline: %w", err)
	// }

	requestBytes, err := proto.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf encryption data request message: %w", err)
	}

	requestPayloadWithLength := make([]byte, 8+len(requestBytes))
	binary.LittleEndian.PutUint64(requestPayloadWithLength, uint64(len(requestBytes)))
	copy(requestPayloadWithLength[8:], requestBytes)
	_, err = s.Write(requestPayloadWithLength)
	if err != nil {
		return nil, fmt.Errorf("failed to write encryption data request to stream: %w", err)
	}

	msgLengthBuffer := make([]byte, 8)
	c := bufio.NewReader(s)
	_, err = c.Read(msgLengthBuffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read encryption data from stream: %w", err)
	}

	// create a buffer with the size of the message and then read until its full
	lengthPrefix := int64(binary.LittleEndian.Uint64(msgLengthBuffer))
	buf := make([]byte, lengthPrefix)

	// read the full message
	_, err = io.ReadFull(c, buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read protobuf encryption data from stream to buffer: %w", err)
	}

	keyData := messages.KeyIVRandomizedFileSegmentsEnvelopeProto{}
	if err := proto.Unmarshal(buf, &keyData); err != nil {
		return nil, fmt.Errorf("failed to unmarshall encryption data from stream: %w", err)
	}

	return &keyData, nil
}

// handleIncomingEncryptionDataTransfer handles incoming encryption data request.
func (d *Protocol) handleIncomingEncryptionDataTransfer(s network.Stream) {
	c := bufio.NewReader(s)
	defer s.Close()

	// read the first 8 bytes to determine the size of the message
	msgLengthBuffer := make([]byte, 8)
	_, err := c.Read(msgLengthBuffer)
	if err != nil {
		log.Errorf("failed to read from handleIncomingEncryptionDataTransfer stream: %s", err.Error())
		return
	}

	// create a buffer with the size of the message and then read until its full
	lengthPrefix := int64(binary.LittleEndian.Uint64(msgLengthBuffer))
	buf := make([]byte, lengthPrefix)

	// read the full message
	_, err = io.ReadFull(c, buf)
	if err != nil {
		log.Errorf("failed to read from handleIncomingEncryptionDataTransfer stream to buffer: %s", err.Error())
		return
	}

	keyIVrequest := messages.KeyIVRequestProto{}
	if err := proto.Unmarshal(buf, &keyIVrequest); err != nil {
		log.Errorf("failed to unmarshall data from handleIncomingEncryptionDataTransfer stream: %s", err.Error())
		return
	}

	contractHashHex := hexutil.Encode(keyIVrequest.ContractHash)
	fileInfo, err := d.contractStore.GetContractFileInfo(contractHashHex, keyIVrequest.FileHash)
	if err != nil {
		log.Errorf("failed to get contract file info in handleIncomingEncryptionDataTransfer: %s", err.Error())
		return
	}

	if fileInfo.ReceivedUnencryptedDataFromFileHoster {
		fileHashHex := hexutil.Encode(fileInfo.FileHash)
		destinationFilePath := filepath.Join(d.downloadDirectory, verifierSubDirectory, contractHashHex, fileHashHex)
		_, _, totalSegmentsToEncrypt, encryptEverySegment := common.FileSegmentsInfo(int(fileInfo.FileSize), d.merkleTreeTotalSegments, d.encryptionPercentage)
		orderedSliceForRawfile := []int{}
		for i := 0; i < totalSegmentsToEncrypt; i++ {
			orderedSliceForRawfile = append(orderedSliceForRawfile, i)
		}

		merkleTreeRandomizedSegments := make([]common.FileBlockHash, len(fileInfo.MerkleTreeNodes))
		for i, v := range fileInfo.MerkleTreeNodes {
			fbh := common.FileBlockHash{
				X: make([]byte, len(v)),
			}
			copy(fbh.X, v)
			merkleTreeRandomizedSegments[i] = fbh
		}

		merkleOfRawSegmentsBeforeEncryption, err := common.HashFileBlockSegments(destinationFilePath, totalSegmentsToEncrypt, orderedSliceForRawfile)
		if err != nil {
			log.Errorf("failed to get file block hashes in handleIncomingEncryptionDataTransfer: %s", err.Error())
			return
		}
		reorderedMerkle, err := common.RetrieveMerkleTreeNodesFromFileWithRawData(encryptEverySegment, fileInfo.RandomSegments, merkleTreeRandomizedSegments, merkleOfRawSegmentsBeforeEncryption)
		if err != nil {
			log.Errorf("failed to retrieve the original order of merkle tree nodes in handleIncomingEncryptionDataTransfer: %s", err.Error())
			return
		}
		merkleOfReorderedMerkle, err := common.GetFileMerkleRootHashFromNodes(reorderedMerkle)
		if err != nil {
			log.Errorf("failed get merkle root hash in handleIncomingEncryptionDataTransfer: %s", err.Error())
			return
		}

		if bytes.Equal(merkleOfReorderedMerkle, fileInfo.MerkleRootHash) {
			err = d.contractStore.SetProofOfTransferVerified(contractHashHex, fileInfo.FileHash, true)
			if err != nil {
				log.Errorf("failed to set proof of transfer verified: %s", err.Error())
				return
			}
		}
	}

	// read again the file info
	fileInfo, _ = d.contractStore.GetContractFileInfo(contractHashHex, keyIVrequest.FileHash)
	if !fileInfo.ProofOfTransferVerified {
		log.Errorf("proof of transfer failed")
		return
	}

	randomizedSegments := make([]int32, len(fileInfo.RandomSegments))
	for i, v := range fileInfo.RandomSegments {
		randomizedSegments[i] = int32(v)
	}

	response := messages.KeyIVRandomizedFileSegmentsEnvelopeProto{
		ContractHash:       keyIVrequest.ContractHash,
		FileHash:           fileInfo.FileHash,
		Key:                fileInfo.Key,
		Iv:                 fileInfo.IV,
		EncryptionType:     int32(fileInfo.EncryptionType),
		RandomizedSegments: randomizedSegments,
	}

	responseBytes, err := proto.Marshal(&response)
	if err != nil {
		log.Errorf("failed to marshal protobuf encryption data in handleIncomingEncryptionDataTransfer message: %s", err.Error())
		return
	}

	responseBytesPayloadWithLength := make([]byte, 8+len(responseBytes))
	binary.LittleEndian.PutUint64(responseBytesPayloadWithLength, uint64(len(responseBytes)))
	copy(responseBytesPayloadWithLength[8:], responseBytes)
	_, err = s.Write(responseBytesPayloadWithLength)
	if err != nil {
		log.Errorf("failed to write encryption key data in handleIncomingEncryptionDataTransfer to stream: %s", err.Error())
		return
	}
}

// SendFileMerkleTreeNodesToVerifier sends the file merkle tree nodes to the verifier.
func (d *Protocol) SendFileMerkleTreeNodesToVerifier(ctx context.Context, verifierID peer.ID, request *messages.MerkleTreeNodesOfFileContractProto) error {
	s, err := d.host.NewStream(ctx, verifierID, ReceiveMerkleTreeProtocolID)
	if err != nil {
		return fmt.Errorf("failed to create new stream to verifier for sending merkle tree nodes: %w", err)
	}
	defer s.Close()

	// future := time.Now().Add(deadlineTimeInSecond * time.Second)
	// err = s.SetDeadline(future)
	// if err != nil {
	// 	return fmt.Errorf("failed to set merkle tree nodes for verifier stream deadline: %w", err)
	// }

	requestBytes, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf merkle tree nodes request message: %w", err)
	}

	requestPayloadWithLength := make([]byte, 8+len(requestBytes))
	binary.LittleEndian.PutUint64(requestPayloadWithLength, uint64(len(requestBytes)))
	copy(requestPayloadWithLength[8:], requestBytes)
	_, err = s.Write(requestPayloadWithLength)
	if err != nil {
		return fmt.Errorf("failed to write merkle tree nodes request to stream: %w", err)
	}

	return nil
}

// SendKeyIVRandomizedFileSegmentsAndDataToVerifier sends the encryption key and iv with the random segments and the unencrypted file segments.
func (d *Protocol) SendKeyIVRandomizedFileSegmentsAndDataToVerifier(ctx context.Context, verifierID peer.ID, filePath string, contractHash string, fileHash []byte) error {
	fileContractInfo, err := d.contractStore.GetContractFileInfo(contractHash, fileHash)
	if err != nil {
		return fmt.Errorf("failed to get contract and file info in sendKeyIVRandomizedFileSegmentsAndDataToVerifier: %w ", err)
	}

	s, err := d.host.NewStream(ctx, verifierID, ReceiveKeyIVRandomizedFileSegmentsAndDataProtocolID)
	if err != nil {
		return fmt.Errorf("failed to create new stream to verifier for sending merkle tree nodes: %w", err)
	}
	defer s.Close()

	// future := time.Now().Add(deadlineTimeInSecond * time.Second)
	// err = s.SetDeadline(future)
	// if err != nil {
	// 	return fmt.Errorf("failed to set merkle tree nodes for verifier stream deadline: %w", err)
	// }

	inputFile, err := os.OpenFile(filePath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer inputFile.Close()

	inputStats, err := inputFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to get stats of input file: %w", err)
	}

	howManySegmentsAllowedForFile, segmentSizeBytes, totalSegmentsToEncrypt, _ := common.FileSegmentsInfo(int(inputStats.Size()), d.merkleTreeTotalSegments, d.encryptionPercentage)
	contractHashBytes, err := hexutil.Decode(contractHash)
	if err != nil {
		return fmt.Errorf("failed to decode contract hash: %w", err)
	}

	randomizedSegments := make([]int32, len(fileContractInfo.RandomSegments))
	for i, v := range fileContractInfo.RandomSegments {
		randomizedSegments[i] = int32(v)
	}

	request := messages.KeyIVRandomizedFileSegmentsEnvelopeProto{
		FileSize:                        uint64(inputStats.Size()),
		ContractHash:                    contractHashBytes,
		FileHash:                        fileHash,
		Key:                             fileContractInfo.Key,
		Iv:                              fileContractInfo.IV,
		MerkleRootHash:                  fileContractInfo.MerkleRootHash,
		EncryptionType:                  int32(fileContractInfo.EncryptionType),
		RandomizedSegments:              randomizedSegments,
		TotalSizeRawUnencryptedSegments: uint64(totalSegmentsToEncrypt) * uint64(segmentSizeBytes),
	}

	requestBytes, err := proto.Marshal(&request)
	if err != nil {
		return fmt.Errorf("failed to marshal protobuf merkle tree nodes request message: %w", err)
	}

	requestPayloadWithLength := make([]byte, 8+len(requestBytes))
	binary.LittleEndian.PutUint64(requestPayloadWithLength, uint64(len(requestBytes)))
	copy(requestPayloadWithLength[8:], requestBytes)
	_, err = s.Write(requestPayloadWithLength)
	if err != nil {
		return fmt.Errorf("failed to write merkle tree nodes request to stream: %w", err)
	}

	err = common.WriteUnencryptedSegments(int(inputStats.Size()), howManySegmentsAllowedForFile, d.encryptionPercentage, fileContractInfo.RandomSegments, inputFile, s)
	if err != nil {
		return fmt.Errorf("failed to write unencrypted data to verifier's stream: %w", err)
	}

	return nil
}

// handleIncomingKeyIVRandomizedFileSegmentsAndData this message is sent from the file hoster to the verifier node
// which contains the metadata and the unencrypted file segments.
func (d *Protocol) handleIncomingKeyIVRandomizedFileSegmentsAndData(s network.Stream) {
	c := bufio.NewReader(s)
	defer s.Close()

	// read the first 8 bytes to determine the size of the message
	msgLengthBuffer := make([]byte, 8)
	_, err := c.Read(msgLengthBuffer)
	if err != nil {
		log.Errorf("failed to read from handleIncomingKeyIVRandomizedFileSegmentsAndData stream: %s", err.Error())
		return
	}

	// create a buffer with the size of the message and then read until its full
	lengthPrefix := int64(binary.LittleEndian.Uint64(msgLengthBuffer))
	buf := make([]byte, lengthPrefix)

	// read the full message
	_, err = io.ReadFull(c, buf)
	if err != nil {
		log.Errorf("failed to read from handleIncomingKeyIVRandomizedFileSegmentsAndData stream to buffer: %s", err.Error())
		return
	}

	keyIVRandomizedFileSegmentsEnvelope := messages.KeyIVRandomizedFileSegmentsEnvelopeProto{}
	if err := proto.Unmarshal(buf, &keyIVRandomizedFileSegmentsEnvelope); err != nil {
		log.Errorf("failed to unmarshall data from handleIncomingKeyIVRandomizedFileSegmentsAndData stream: %s", err.Error())
		return
	}

	contractHash := hexutil.Encode(keyIVRandomizedFileSegmentsEnvelope.ContractHash)

	fileContract, err := d.contractStore.GetContract(contractHash)
	if err != nil {
		log.Errorf("failed to get contract in handleIncomingKeyIVRandomizedFileSegmentsAndData: %s", err.Error())
		return
	}

	publicKeyFileHoster, err := ffgcrypto.PublicKeyFromBytes(fileContract.FileHosterResponse.PublicKey)
	if err != nil {
		log.Errorf("failed to get the public key of the file hoster: %s", err.Error())
		return
	}

	if !verifyConnection(publicKeyFileHoster, s.Conn().RemotePublicKey()) {
		log.Error("malicious request from host which is not file hoster")
		return
	}

	randomizedSegments := make([]int, len(keyIVRandomizedFileSegmentsEnvelope.RandomizedSegments))
	for i, v := range keyIVRandomizedFileSegmentsEnvelope.RandomizedSegments {
		randomizedSegments[i] = int(v)
	}

	err = d.contractStore.SetKeyIVEncryptionTypeRandomizedFileSegments(contractHash, keyIVRandomizedFileSegmentsEnvelope.FileHash, keyIVRandomizedFileSegmentsEnvelope.Key, keyIVRandomizedFileSegmentsEnvelope.Iv, keyIVRandomizedFileSegmentsEnvelope.MerkleRootHash, common.EncryptionType(keyIVRandomizedFileSegmentsEnvelope.EncryptionType), randomizedSegments, keyIVRandomizedFileSegmentsEnvelope.FileSize)
	if err != nil {
		log.Errorf("failed to update key, iv and random segments of file contract: %s", err.Error())
		return
	}

	contractHashHex := hexutil.Encode(keyIVRandomizedFileSegmentsEnvelope.ContractHash)
	err = common.CreateDirectory(filepath.Join(d.downloadDirectory, verifierSubDirectory, contractHashHex))
	if err != nil {
		log.Errorf("failed to created contract directory: %s", err.Error())
		return
	}

	fileHashHex := hexutil.Encode(keyIVRandomizedFileSegmentsEnvelope.FileHash)
	destinationFilePath := filepath.Join(d.downloadDirectory, verifierSubDirectory, contractHashHex, fileHashHex)
	destinationFile, err := os.OpenFile(destinationFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		log.Errorf("failed to open a file for downloading its content from hoster: %s", err.Error())
		return
	}
	defer destinationFile.Close()

	buf = make([]byte, bufferSize)
	totalFileBytesReceived := uint64(0)
	for totalFileBytesReceived != keyIVRandomizedFileSegmentsEnvelope.TotalSizeRawUnencryptedSegments {
		n, err := s.Read(buf)
		if n > 0 {
			wroteN, err := destinationFile.Write(buf[:n])
			if wroteN != n || err != nil {
				log.Errorf("failed to write the total content of buffer (buf: %d, output: %d) to output file: %s", n, wroteN, err.Error())
				return
			}
			totalFileBytesReceived += uint64(wroteN)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Errorf("fialed to read file content to buffer: %s", err.Error())
			return
		}
	}

	err = d.contractStore.SetReceivedUnencryptedDataFromFileHoster(contractHash, keyIVRandomizedFileSegmentsEnvelope.FileHash, true)
	if err != nil {
		log.Errorf("failed to set received unencrypted data from file hoster: %s", err.Error())
		return
	}
}

// handleIncomingMerkleTreeNodes handles incoming merkle tree nodes from a node.
// this protocol handler is used by a verifier.
func (d *Protocol) handleIncomingMerkleTreeNodes(s network.Stream) {
	c := bufio.NewReader(s)
	defer s.Close()

	// read the first 8 bytes to determine the size of the message
	msgLengthBuffer := make([]byte, 8)
	_, err := c.Read(msgLengthBuffer)
	if err != nil {
		log.Errorf("failed to read from handleIncomingMerkleTreeNodes stream: %s", err.Error())
		return
	}

	// create a buffer with the size of the message and then read until its full
	lengthPrefix := int64(binary.LittleEndian.Uint64(msgLengthBuffer))
	buf := make([]byte, lengthPrefix)

	// read the full message
	_, err = io.ReadFull(c, buf)
	if err != nil {
		log.Errorf("failed to read from handleIncomingMerkleTreeNodes stream to buffer: %s", err.Error())
		return
	}

	merkleTreeNodesOfFileMessage := messages.MerkleTreeNodesOfFileContractProto{}
	if err := proto.Unmarshal(buf, &merkleTreeNodesOfFileMessage); err != nil {
		log.Errorf("failed to unmarshall data from handleIncomingMerkleTreeNodes stream: %s", err.Error())
		return
	}

	contractHash := hexutil.Encode(merkleTreeNodesOfFileMessage.ContractHash)

	fileContract, err := d.contractStore.GetContract(contractHash)
	if err != nil {
		log.Errorf("failed to get contract in handleIncomingMerkleTreeNodes: %s", err.Error())
		return
	}

	publicKeyFileRequester, err := ffgcrypto.PublicKeyFromBytes(fileContract.FileRequesterNodePublicKey)
	if err != nil {
		log.Errorf("failed to get the public key of the file requester: %s", err.Error())
		return
	}

	if !verifyConnection(publicKeyFileRequester, s.Conn().RemotePublicKey()) {
		log.Error("malicious request from downloader")
		return
	}

	err = d.contractStore.SetMerkleTreeNodes(contractHash, merkleTreeNodesOfFileMessage.FileHash, merkleTreeNodesOfFileMessage.MerkleTreeNodes)
	if err != nil {
		log.Errorf("failed to update merkle tree nodes for a file contract: %s", err.Error())
		return
	}
}

// RequestFileTransfer requests a file download from the file hoster.
// Request is initiated from the downloader peer.
// TODO: handle network failure and resumable file transfer.
func (d *Protocol) RequestFileTransfer(ctx context.Context, fileHosterID peer.ID, request *messages.FileTransferInfoProto) (string, error) {
	s, err := d.host.NewStream(ctx, fileHosterID, FileTransferProtocolID)
	if err != nil {
		return "", fmt.Errorf("failed to create new file download stream to file hoster: %w", err)
	}
	defer s.Close()

	// future := time.Now().Add(deadlineTimeInSecond * time.Second)
	// err = s.SetDeadline(future)
	if err != nil {
		return "", fmt.Errorf("failed to set file transfer stream deadline: %w", err)
	}

	requestBytes, err := proto.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal protobuf file transfer request message: %w", err)
	}

	requestPayloadWithLength := make([]byte, 8+len(requestBytes))
	binary.LittleEndian.PutUint64(requestPayloadWithLength, uint64(len(requestBytes)))
	copy(requestPayloadWithLength[8:], requestBytes)
	_, err = s.Write(requestPayloadWithLength)
	if err != nil {
		return "", fmt.Errorf("failed to write file transfer request to stream: %w", err)
	}

	contractHashHex := hexutil.Encode(request.ContractHash)
	err = common.CreateDirectory(filepath.Join(d.downloadDirectory, contractHashHex))
	if err != nil {
		return "", fmt.Errorf("failed to created contract directory: %w", err)
	}

	fileHashHex := hexutil.Encode(request.FileHash)
	destinationFilePath := filepath.Join(d.downloadDirectory, contractHashHex, fileHashHex)
	destinationFile, err := os.OpenFile(destinationFilePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to open a file for downloading its content from hoster: %w", err)
	}
	defer destinationFile.Close()

	buf := make([]byte, bufferSize)
	totalFileBytesTransfered := uint64(0)
	for totalFileBytesTransfered != request.FileSize {
		n, err := s.Read(buf)
		if n > 0 {
			wroteN, err := destinationFile.Write(buf[:n])
			if wroteN != n || err != nil {
				return "", fmt.Errorf("failed to write the total content of buffer (buf: %d, output: %d) to output file: %w", n, wroteN, err)
			}
			totalFileBytesTransfered += uint64(wroteN)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return "", fmt.Errorf("fialed to read file content to buffer: %w", err)
		}
	}

	return destinationFilePath, nil
}

// handleIncomingFileTransfer handles an incoming file transfer initiated from file downloader towards file hoster node.
func (d *Protocol) handleIncomingFileTransfer(s network.Stream) {
	c := bufio.NewReader(s)
	defer s.Close()

	// read the first 8 bytes to determine the size of the message
	msgLengthBuffer := make([]byte, 8)
	_, err := c.Read(msgLengthBuffer)
	if err != nil {
		log.Errorf("failed to read from handleIncomingFileTransfer stream: %s", err.Error())
		return
	}

	// create a buffer with the size of the message and then read until its full
	lengthPrefix := int64(binary.LittleEndian.Uint64(msgLengthBuffer))
	buf := make([]byte, lengthPrefix)

	// read the full message
	_, err = io.ReadFull(c, buf)
	if err != nil {
		log.Errorf("failed to read from handleIncomingFileTransfer stream to buffer: %s", err.Error())
		return
	}

	fileTransferRequest := messages.FileTransferInfoProto{}
	if err := proto.Unmarshal(buf, &fileTransferRequest); err != nil {
		log.Errorf("failed to unmarshall data from handleIncomingFileTransfer stream: %s", err.Error())
		return
	}

	contractHash := hexutil.Encode(fileTransferRequest.ContractHash)
	fileContractInfo, err := d.contractStore.GetContractFileInfo(contractHash, fileTransferRequest.FileHash)
	if err != nil {
		log.Errorf("failed to get contract and file info in handleIncomingFileTransfer: %s", err.Error())
		return
	}

	downloadContract, err := d.contractStore.GetContract(contractHash)
	if err != nil {
		log.Errorf("failed to get contract in handleIncomingFileTransfer: %s", err.Error())
		return
	}

	publicKeyFileRequester, err := ffgcrypto.PublicKeyFromBytes(downloadContract.FileRequesterNodePublicKey)
	if err != nil {
		log.Errorf("failed to get the public key of the file requester: %s", err.Error())
		return
	}

	if !verifyConnection(publicKeyFileRequester, s.Conn().RemotePublicKey()) {
		log.Error("malicious request from downloader")
		return
	}

	fileHashHex := hexutil.EncodeNoPrefix(fileTransferRequest.FileHash)
	fileMetadata, err := d.storage.GetFileMetadata(fileHashHex)
	if err != nil {
		log.Errorf("failed to get file metadata from storage engine in handleIncomingFileTransfer: %s", err.Error())
		return
	}

	input, err := os.Open(fileMetadata.FilePath)
	if err != nil {
		log.Errorf("failed to open file for encryption and streaming in handleIncomingFileTransfer: %s", err.Error())
		return
	}

	encryptor, err := common.NewEncryptor(fileContractInfo.EncryptionType, fileContractInfo.Key, fileContractInfo.IV)
	if err != nil {
		log.Errorf("failed to setup encryptor in handleIncomingFileTransfer: %s", err.Error())
		return
	}

	// write to the stream the content of the input file while encrypting and shuffling its segments.
	err = common.EncryptWriteOutput(int(fileMetadata.Size), d.merkleTreeTotalSegments, d.encryptionPercentage, fileContractInfo.RandomSegments, input, s, encryptor)
	if err != nil {
		log.Errorf("failed to encryptWriteOutput in handleIncomingFileTransfer: %s", err.Error())
		return
	}

	err = input.Close()
	if err != nil {
		return
	}
}

func verifyConnection(from, to crypto.PubKey) bool {
	return from.Equals(to)
}