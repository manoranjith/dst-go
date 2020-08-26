# Glossary
| Terms          | Definition   |
| :------------- | :----------: |
|Account - Onchain Account|Onchain account represents an account corresponding to an onchain address. It is used for signing transactions and funding ledger channels.|
|Account - Offchain Account|Account represents an account stored in a wallet corresponding to an address. It is used to sign messages.|
|Alias| Alias is used to reference a peer in all API calls.|
|Balances|Balances is the amount held by the Peer corresponding to the Alias.|
|Chain Address|Chain address is the address of the default blockchain node used by the perun node.|
|Comm Address|Address of the peer for off-chain communication.|
|Comm Type|Type of communication protocol supported by the peer for off-chain communication.|
|Contact Types|Contact Types is the contacts Provider backends supported by the node.|
|Currency|Currency used for specifying the amount.|
|Final|Final indicates if this is a final update. Channel will be closed once a final update is accepted.|
|ID- Channel ID|Channel ID is the unique ID to represent a channel, derived from its Parameters.|
|ID- Proposal ID|Proposal ID is the unique ID of the channel proposal.|
|ID- Session ID|Session ID is the unique ID of the session.|
|Identity - Network Address|Address used for establishing physical network connection with other participants. It depends on the network protocol. For the TCP/IP adapter used in current implementation, it is an IP address.|
|Identity - Off-chain Address|Address used for participating in a channel. It is used for signing for state updates. It is often an ephemeral address. Hence the same participant can have different addresses in different channels.|
|Identity - On-chain Address|Address used for funding the channels and the on-chain transactions.|
|Identity - Perun Address|Address that represents the permanent identity of a user in the perun network. The user can authenticate itself by signing with this address. It is independent of any blockchain.The physical connection is established using the associated network address.|
|Node|Node is a running instance of a program, that enables a user to establish and manage state channels via the Perun protocol. It offers an RPC interface for the user.|
|Payee|Payee is the alias of payee. Self indicates own user.|
|PeerAlias|PeerAlias is the alias of peer with whom channel should be opened.|
|Protocol - Perun|The protocol used for creating and transacting on state channels, as proposed in the Perun papers.|
|Protocol - Transport Layer|Standard network transport layer protocols such as TCP/IP or Websocket.|
|Protocol - Wire|Communication protocol for handling off-chain messages among peers in the Perun network. It is independent of the transport layer.|
|Timeout|Timeout is the time (in unix format) before which response should be sent.|
|Version|Version indicates the current version of the state in the channel. The initial state has version 0. All state updates increase the version by 1.|