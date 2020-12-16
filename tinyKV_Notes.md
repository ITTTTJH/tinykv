# Project1

project1主要是实现standalone_storage 中的write函数和Reader函数 以及StorageReader接口的相关函数和server中的Rawget Rawdelete Rawput RawScan函数

对badger数据库的具体操作基本已在engine_util包中实现，只需根据不同操作调用engine_util包中的相应函数即可

# Project2

## Project2A

基本是面向测试编程，根据测试函数的逻辑和raft论文的描述来调整具体实现

### 2AA

**实现领导人选举：**具体实现中通过tick函数表示逻辑时钟，调用一次则将内部逻辑时钟提前一个刻度，从而驱动选举超时或心跳超时。在step函数中处理Raft消息，实现各个状态下的step函数中与选主相关的部分。

### 2AB

**实现日志复制：**在step函数中处理MsgAppend消息并根据raft消息中的信息和自身信息回复相应的MsgAppendResponse。

### 2AC

实现Raft模块与上层交互的相关函数

Raft模块通过返回一个ready结构体将要发送的消息、要提交的日志条目、当前状态返回给上层进行处理。

**a**．ready

ready函数填充ready结构体相应数据，返回ready结构体 

**b.HasReady**

判断是否rn.Raft.RaftLog.unstableEntries()、rn.Raft.RaftLog.nextEnts()、rn.Raft.msgs是否都为空，若都为空返回false

**c.Advance**

通过rawnode里的ready中的Entrie等数据来修改raft中的applied index, stabled log index等数据

## Project2B

服务端运行流程大致如下：

1客户端调用RawGet / RawPut / RawDelete / RawScan RPC

2RPC处理程序调用RaftStorage相关方法

3RaftStorage 向Raftstore发送Raft命令请求，并等待响应

4RaftStore 将Raft命令作为Raft日志进行处理

5Raft模块附加日志，并通过 PeerStorage持久化到本地数据库

6Raft模块提交日志

7Raft worker在处理Raft模块返回的ready结构体时执行Raft命令，并通过调callback返回相应的响应

8RaftStorage 接收返回的响应并返回到RPC处理程序

9RPC处理程序将RPC响应返回给客户端。

要实现的部分主要是处理Raft命令的proposeRaftCommand函数和处理Raft返回的ready结构体的HandleRaftReady函数

## Project2C

Raft 的日志在正常操作中不断的增长，但是在实际的系统中，日志不能无限制的增长。随着日志不断增长，他会占用越来越多的空间，花费越来越多的时间来重置。如果没有一定的机制去清除日志里积累的陈旧的信息，那么会带来可用性问题。因此需定期对日志进行压缩，日志压缩流程大致如下：

1raftlog-gc worker定时检测是否需要进行raftlog-gc

2若需要进行raftlog-gc则发送admin command CompactLogRequest

3proposeRaftCommand中调用propose进行处理

4HandleRaftReady中在apply命令时若发现该命令是CompactLogRequest，则修改TruncatedState并发送一个raftLogGcTask到raftLogGCTaskSender，在另一个线程中对raftdb中的日志进行压缩

由于日志压缩可能导致某些Follower没有完全接收到Leader所有的日志条目而Leader的一些日志条目已经被压缩了，因此此时Leader需要向Follower发送快照来使Follower日志同步，需要增加处理快照的相关逻辑，处理快照的大致流程如下：

1Raft中若需要发送Snapshot则调用Storage.Snapshot()生成日志，然后作为raft消息发送

2接收方收到Snapshot调用handleSnapshot进行处理，修改相关字段，将Snapshot保存到pendingSnapshot中

3在上层调用ready时将pendingSnapshot包装至ready中返回给上层

4SaveReadyState中若检测到有Snapshot则调用ApplySnapshot更新相关信息进行处理

# Project3

## Project3A

**在Raft模块增加领导者更改和成员更改**

在Raft模块增加MsgTransferLeader，EntryConfChange，MsgHup相关处理逻辑

## Project3B（有bug）

**在**RaftStore**中增加领导者更改、成员更改、区域分割**

**（**1**）领导者更改**

​    proposeRaftCommand中直接调用rawnode的TransferLeader并返回相应的response即可

**（**2**）成员更改**

proposeRaftCommand中调用rawnode的ProposeConfChange

​    handleraftready中更新confver、ctx.storeMeta、PeerCache等相关信息，调用rawnode的ApplyConfChange

**（**3**）区域分割**

​    Region中维护了该region的start key和end key，当Region中的数据过多时应对该Region进行分割，划分为两个Region从而提高并发度，执行区域分割的大致流程如下：

​	1split checker检测是否需要进行Region split，若需要则生成split key并发送相应命令

​    2proposeRaftCommand执行propose操作

​    3提交后在HandleRaftReady对该命令进行处理：

​       检测split key是否在start key 和end key之间，不满足返回

​       创建新region 原region和新region key范围为start~split，split~end

​       更新d.ctx.storeMeta中的regionranges和regions

​       为新region创建peer，调insertPeerCache插入peer信息

​       发送MsgTypeStart消息

4执行get命令时检测请求的key是否在该region中，若不在该region中返回相应错误

## Project3C

​	kv数据库中所有数据被分为几个Region，每个Region包含多个副本，调度程序负责调度每个副本的位置以及Store中Region的个数以避免Store中有过多Region。

​    为实现调度功能，region应向调度程序定期发送心跳，使调度程序能更新相关的信息从而能实现正确的调度，processRegionHeartbea函数实现处理region发送的心跳，该函数通过regioninfo更新raftcluster中的相关信息。

​	为避免一个store中有过多Region而影响性能，要对Store中的Region进行调度，通过Schedule函数实现

# Project4

​	面向测试，结合指导文档

​	4A 实现结构体MvccTxn的相关函数

​    4B 实现server.go中的KvGet, KvPrewrite, KvCommit函数

​    4C 实现server.go中的KvScan, KvCheckTxnStatus, KvBatchRollback,KvResolveLock函数