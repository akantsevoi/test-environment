# gateway flow
- gets request from the client
- sends request to maroon { body, rangeIndex, offset }
maroonNode
- increments value in the vector <(rangeIndex, offset)> if it's the last offset!!
- when range is closer to the end - requests the next one

# maroonNode flow:
- stores two offset vectors:
	- commited <(1,10), (2,8), (3,5)...>
  - uncommitedMajority <(1,10), (2,13), (3,7)...>
	- uncommitedLocal <(1,10), (2,14), (3,7)...>
    - uncommitedLocal is populated from the transactions spread by gateway and gossipped by other nodes
- periodically publishes it's uncommited vector to other nodes
  - by this it confirms that node stored transaction
- "leader" listens to the `uncommited` updates from the other nodes and updates uncommitedMajority vector
  - for example it might look like:
```
  NL <(1,10), (2,14), (3,7)...>
	N2 <(1,10), (2,12), (3,6)...>
	N3 <(1,11), (2,13), (3,7)...>
```
 then the uncommitedMajority will be `<(1,10), (2,13), (3,7)...>`
- periodically takes uncommitedMajority and puts it to etcd
  - "/maroon/tn" = <(1,10), (2,13), (3,7)...>
- when nodes get new uncommitedMajority update from etcd
  - they update their commited vectors to the latest
	- commited <(1,10), (2,8), (3,5)...> -> <(1,10), (2,13), (3,7)...>
	- And since key range space is deterministic and deterministically orderable - they can sort transactions the same way.
