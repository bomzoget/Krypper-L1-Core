# Krypper Chain - Blockchain Node

## Overview
Krypper Chain is a blockchain implementation written in Go featuring a three-tier consensus mechanism with Proposers (Tier 1), Validators (Tier 2), and Witnesses (Tier 3). The project includes a full blockchain node with RPC server, P2P networking, and CLI tools for interaction.

## Recent Changes (December 2, 2025)
- Fixed StateDB implementation with missing methods (GetBalance, GetNonce, StateRoot, Snapshot, RevertToSnapshot, CommitSnapshot, SubBalance, AddBalance, IncrementNonce)
- Resolved duplicate type definitions between types.go and separate type files (address.go, block.go)
- Added missing methods to Executor (SetCurrentHeader, SetCoinbase) and Block (ValidateBasic)
- Fixed ValidatorVote field references (Voter instead of Validator, Block instead of BlockHash)
- Configured project to use port 8000 instead of 8545 for RPC server
- Installed Go 1.24 toolchain and all dependencies
- Set up blockchain node workflow with console output

## Project Architecture

### Core Components

#### Main Node (`main.go`)
- Entry point for the blockchain node
- Initializes StateDB, mempool, executor, and blockchain
- Creates deterministic genesis block
- Starts RPC server on port 8000
- Runs background mining loop

#### Type System (`types/`)
- **Account** (`account.go`): User account structure with balance, nonce, code hash, storage root, and frozen status
- **Address** (`address.go`): 20-byte EVM-compatible address type
- **Block** (`block.go`): Block structure with header and transactions, supports three-tier consensus
- **Blockchain** (`blockchain.go`): Chain management with validation and state transitions
- **Transaction** (`transaction.go`): Transaction structure with signing and verification
- **StateDB** (`statedb.go`): In-memory state management with snapshot/revert capability
- **Executor** (`executor.go`): Transaction execution with tier-based reward distribution
- **Validator** (`validator.go`): Tier-2 validator vote system
- **Witness** (`witness.go`): Tier-3 mobile witness support
- **Mempool** (`mempool.go`): Transaction pool management

#### Node Logic (`node/`)
- Mining loop with 5-second block time
- Witness and validator vote management
- Block creation with three-tier participant selection
- Dry-run execution with state snapshots

#### RPC Server (`rpc/`)
- HTTP JSON-RPC endpoints on port 8000:
  - `/tx/send` - Submit transactions
  - `/account/balance` - Query account balance
  - `/chain/head` - Get current chain head
  - `/witness/submit` - Submit Tier-3 witness
  - `/validator/vote` - Submit Tier-2 validator vote

#### P2P Networking (`p2p/`)
- Peer discovery and management
- Block synchronization
- Message broadcasting

#### Command-line Tools (`cmd/`)
- **krypcli**: Wallet management, balance queries, transaction sending
- **validator**: Tier-2 validator node
- **krypmobile**: Tier-3 mobile witness/miner

### Three-Tier Consensus Model

1. **Tier 1 - Proposers**: Main block producers (70% of fees)
2. **Tier 2 - Validators**: Block validators (20% of fees)
3. **Tier 3 - Witnesses**: Mobile witnesses (5% of fees)
4. **Pool**: Reserve fund (5% of fees)

## Running the Project

### Blockchain Node
The blockchain node runs automatically via the workflow system:
```bash
go run main.go
```

Optional flags:
- `-port` - RPC port (default: 8000)
- `-peers` - Comma-separated peer URLs

### CLI Tools

#### Create a new wallet:
```bash
go run cmd/krypcli/main.go new
```

#### Check balance:
```bash
go run cmd/krypcli/main.go balance -addr 0x... -rpc http://localhost:8000
```

#### Send transaction:
```bash
go run cmd/krypcli/main.go send -priv HEX -to ADDRESS -amount WEI -rpc http://localhost:8000
```

### Building
```bash
# Build main node
go build -o krypper-node main.go

# Build CLI
go build -o krypcli cmd/krypcli/main.go

# Build validator
go build -o validator cmd/validator/main.go

# Build mobile witness
go build -o krypmobile cmd/krypmobile/main.go
```

## Configuration

### Chain Configuration
- Chain ID: 1
- Block Time: 5 seconds
- Gas Limit: 30,000,000
- Genesis Supply: 1,000,000 tokens (18 decimals)

### Fee Distribution
- Proposer (Tier 1): 70%
- Validator (Tier 2): 20%
- Witness (Tier 3): 5%
- Reserve Pool: 5%

## Technical Details

### State Management
- In-memory StateDB with snapshot/revert capability
- Merkle root computation for state verification
- Account-based model with balance, nonce, code hash, and storage root

### Transaction Flow
1. Transaction submitted to mempool via RPC
2. Signature verification and sender recovery
3. Mining loop selects transactions for new block
4. Dry-run execution with snapshot
5. State root computation
6. Block finalization with tier-based rewards
7. State commit and block indexing

### Security
- ECDSA signature verification
- Chain ID for replay protection
- Nonce-based transaction ordering
- Gas price and limit validation

## Dependencies
- Go 1.22+
- github.com/ethereum/go-ethereum (crypto utilities)
- github.com/btcsuite/btcd/btcec/v2
- github.com/decred/dcrd/dcrec/secp256k1/v4
- github.com/holiman/uint256

## Current Status
- ✅ Core blockchain implementation complete
- ✅ RPC server running on port 8000
- ✅ Genesis block created and validated
- ✅ Mining loop active
- ✅ Transaction execution with tier-based rewards
- ✅ Snapshot/revert state management
- ✅ CLI tools for wallet and transaction management
