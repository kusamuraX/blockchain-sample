package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

// Transaction is Transaction data
type Transaction struct {
	Sender     string
	Receiver   string
	TransValue float64
}

// Block is blockchain one data structure
type Block struct {
	Timestamp     int64
	Data          *Transaction
	PrevBlockHash []byte
	Hash          []byte
	Nonce         int
}

// SetHash is Block function.
// this is generate hash
func (b *Block) SetHash() {
	dataBuf := &bytes.Buffer{}
	binary.Write(dataBuf, binary.LittleEndian, b.Data)
	dataByte := dataBuf.Bytes()[:]
	timestamp := []byte(strconv.FormatInt(b.Timestamp, 10))
	headers := bytes.Join([][]byte{b.PrevBlockHash, dataByte, timestamp}, []byte{})
	hash := sha256.Sum256(headers)
	b.Hash = hash[:]
}

// ProofOfWork is PoW algorithm structure
type ProofOfWork struct {
	block  *Block
	target *big.Int
}

// prepare is PoW prepare data
func (pow *ProofOfWork) prepare(nonce int) []byte {
	dataBuf := &bytes.Buffer{}
	binary.Write(dataBuf, binary.LittleEndian, pow.block.Data)
	dataByte := dataBuf.Bytes()[:]
	data := bytes.Join([][]byte{
		pow.block.PrevBlockHash,
		dataByte,
		IntToHex(pow.block.Timestamp),
		IntToHex(int64(targetBits)),
		IntToHex(int64(nonce)),
	}, []byte{})
	return data
}

// IntToHex is int to hex byte array
func IntToHex(n int64) []byte {
	return []byte(strconv.FormatInt(n, 16))
}

const maxNonce = math.MaxInt64

// Run is run proof of work
func (pow *ProofOfWork) Run() (int, []byte) {
	nonce := make(chan int)
	hashRcv := make(chan [32]byte)
	closeCh := make(chan struct{})

	go getHash(closeCh, nonce, pow, hashRcv)
	// fmt.Print("\n\n")

	hash := <-hashRcv
	currentNonce := <-nonce
	fmt.Println(currentNonce, hash)
	close(closeCh)
	// fmt.Println(hash)
	return currentNonce, hash[:]
}

const threadCnt int = 10

// GetHash is generate hash
func getHash(closeCh chan struct{}, nonce chan int, pow *ProofOfWork, hashRcv chan [32]byte) {

	var once sync.Once
	div := maxNonce / threadCnt
	for index := 0; index < threadCnt; index++ {
		nonceIndex := div * index
		if nonceIndex < 0 {
			nonceIndex = maxNonce
		}
		go getHashChild(closeCh, nonce, pow, hashRcv, nonceIndex, &once)
	}
}

func getHashChild(closeCh chan struct{}, nonce chan int, pow *ProofOfWork, hashRcv chan [32]byte, nonceIndex int, once *sync.Once) {

	defer func() {
		once.Do(func() {
			close(hashRcv)
			close(nonce)
		})
	}()

	var hash [32]byte
	var hashInt big.Int
	for currentNonce := nonceIndex; nonceIndex < (nonceIndex + (maxNonce / threadCnt)); currentNonce++ {
		select {
		case <-closeCh:
			return
		default:
			data := pow.prepare(currentNonce)
			hash = sha256.Sum256(data)
			hashInt.SetBytes(hash[:])
			if hashInt.Cmp(pow.target) == -1 {
				select {
				case <-closeCh:
					return
				default:
					hashRcv <- hash
					nonce <- currentNonce
					return
				}
			}
		}
	}
}

const targetBits = 24

// NewProofOfWork is generate PoW
func NewProofOfWork(b *Block) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-targetBits))
	pow := &ProofOfWork{b, target}
	return pow
}

// Blockchain is blockchain data structure
type Blockchain struct {
	blocks []*Block
}

// NewBlock is genarate new block
func NewBlock(data *Transaction, prevBloackHash []byte) (block *Block) {
	block = &Block{time.Now().Unix(), data, prevBloackHash, []byte{}, 0}
	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()

	block.Hash = hash[:]
	block.Nonce = nonce

	return
}

// AddBlock is Blockchain function
// this is add block
func (bc *Blockchain) AddBlock(data *Transaction) {
	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	bc.blocks = append(bc.blocks, newBlock)
}

// NewGenesisBlock is create genesis block
func NewGenesisBlock() *Block {
	rand.Seed(time.Now().UnixNano())
	genesis := &Transaction{"genesis", "genesis", rand.Float64()}
	return NewBlock(genesis, []byte{})
}

// NewBlockChain is init Blockchain
func NewBlockChain() *Blockchain {
	return &Blockchain{[]*Block{NewGenesisBlock()}}
}

func main() {
	bc := NewBlockChain()

	ts1 := &Transaction{"Alice", "Bob", 1.0}
	ts2 := &Transaction{"Bob", "Alice", 1.0}
	bc.AddBlock(ts1)
	bc.AddBlock(ts2)
	printBlockchain(bc)
}

func printBlockchain(bc *Blockchain) {
	for _, block := range bc.blocks {
		fmt.Printf("Prev hash: %x\n", block.PrevBlockHash)
		fmt.Printf("     Data: %s to %s (%f)\n", block.Data.Sender, block.Data.Receiver, block.Data.TransValue)
		fmt.Printf("     Hash: %x\n", block.Hash)
		fmt.Println()
	}
}
