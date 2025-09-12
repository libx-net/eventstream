module kafka-adapter-example

go 1.23.0

require libx.net/eventstream v0.0.0

require (
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/panjf2000/ants/v2 v2.9.0 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	github.com/segmentio/kafka-go v0.4.49 // indirect
)

replace libx.net/eventstream => ../../
