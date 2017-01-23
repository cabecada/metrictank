package notifierKafka

import (
	"flag"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/raintank/metrictank/cluster"
	"github.com/raintank/metrictank/kafka"
	"github.com/raintank/metrictank/stats"
	"github.com/rakyll/globalconf"
)

var Enabled bool
var brokerStr string
var brokers []string
var topic string
var offsetStr string
var dataDir string
var config *sarama.Config
var offsetDuration time.Duration
var offsetCommitInterval time.Duration
var partitionStr string
var partitions []int32
var partitioner *cluster.KafkaPartitioner
var partitionScheme string

// metric cluster.notifier.kafka.messages-published is a counter of messages published to the kafka cluster notifier
var messagesPublished = stats.NewCounter32("cluster.notifier.kafka.messages-published")

// metric cluster.notifier.kafka.message_size is the sizes seen of messages through the kafka cluster notifier
var messagesSize = stats.NewMeter32("cluster.notifier.kafka.message_size", false)

func init() {
	fs := flag.NewFlagSet("kafka-cluster", flag.ExitOnError)
	fs.BoolVar(&Enabled, "enabled", false, "")
	fs.StringVar(&brokerStr, "brokers", "kafka:9092", "tcp address for kafka (may be given multiple times as comma separated list)")
	fs.StringVar(&topic, "topic", "metricpersist", "kafka topic")
	fs.StringVar(&partitionStr, "partitions", "*", "kafka partitions to consume. use '*' or a comma separated list of id's. This should match the partitions used for kafka-mdm-in")
	fs.StringVar(&partitionScheme, "partition-scheme", "bySeries", "method used for partitioning metrics. This should match the settings of tsdb-gw.  (byOrg|bySeries)")
	fs.StringVar(&offsetStr, "offset", "last", "Set the offset to start consuming from. Can be one of newest, oldest,last or a time duration")
	fs.StringVar(&dataDir, "data-dir", "", "Directory to store partition offsets index")
	fs.DurationVar(&offsetCommitInterval, "offset-commit-interval", time.Second*5, "Interval at which offsets should be saved.")
	globalconf.Register("kafka-cluster", fs)
}

func ConfigProcess(instance string) {
	if !Enabled {
		return
	}
	var err error
	switch offsetStr {
	case "last":
	case "oldest":
	case "newest":
	default:
		offsetDuration, err = time.ParseDuration(offsetStr)
		if err != nil {
			log.Fatal(4, "kafka-cluster: invalid offest format. %s", err)
		}
	}
	brokers = strings.Split(brokerStr, ",")

	config = sarama.NewConfig()
	config.ClientID = instance + "-cluster"
	config.Version = sarama.V0_10_0_0
	config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	config.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Return.Successes = true
	err = config.Validate()
	if err != nil {
		log.Fatal(2, "kafka-cluster invalid consumer config: %s", err)
	}

	partitioner, err = cluster.NewKafkaPartitioner(partitionScheme)
	if err != nil {
		log.Fatal(4, "kafka-cluster: failed to initialize partitioner. %s", err)
	}

	if partitionStr != "*" {
		parts := strings.Split(partitionStr, ",")
		for _, part := range parts {
			i, err := strconv.Atoi(part)
			if err != nil {
				log.Fatal(4, "kafka-cluster: could not parse partition %q. partitions must be '*' or a comma separated list of id's", part)
			}
			partitions = append(partitions, int32(i))
		}
	}
	// validate our partitions
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		log.Fatal(4, "kafka-cluster failed to create client. %s", err)
	}
	defer client.Close()

	availParts, err := kafka.GetPartitions(client, []string{topic})
	if err != nil {
		log.Fatal(4, "kafka-cluster: %s", err.Error())
	}
	if partitionStr == "*" {
		partitions = availParts
	} else {
		missing := kafka.DiffPartitions(partitions, availParts)
		if len(missing) > 0 {
			log.Fatal(4, "kafka-cluster: configured partitions not in list of available partitions. missing %v", missing)
		}
	}
}