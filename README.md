
# Samproxy - the Honeycomb Sampling Proxy

![samproxy](https://user-images.githubusercontent.com/1476820/47527709-0e185f00-d858-11e8-8e66-4fd5294d1918.png)

**Alpha Release** This is the initial draft. Please expect and help find bugs! :)  samproxy [![Build Status](https://circleci.com/gh/honeycombio/samproxy.svg?style=shield)](https://circleci.com/gh/honeycombio/samproxy)

## Purpose

Samproxy is a trace-aware sampling proxy. It collects spans emitted by your applications together in to traces and examines them as a whole in order to use all the information present about how your service handled a given request to make an intelligent decision about how to sample the trace - whether to keep or discard a specific trace. Buffering the spans together let you use fields that might be present in different spans together to influence the sampling decision. It might be that you have the HTTP status code available in the root span, but you want to use other information like whether this request was served from cache to influence sampling.

## Setting up Samproxy

Samproxy is designed to sit within your infrastructure where all sources of Honeycomb events (aka spans if you're doing tracing) can reach it. A standard deployment has a cluster of servers running Samproxy accessible via a load balancer. Samproxy instances must be able to communicate with each other to concentrate traces on single servers.

Within your application (or other Honeycomb event sources) you would configure the `API Host` to be http(s)://load-balancer/. Everything else remains the same (api key, dataset name, etc. - all that lives with the originating client).

### Minimum configuration

The Samproxy cluster should have at least 2 servers with 2GB RAM and access to 2 cores each.

Additional RAM and CPU can be used by increasing configuration values to have a larger `CacheCapacity`. The cluster should be monitored for panics caused by running out of memory and scaled up (with either more servers or more RAM per server) when they occur.

### Builds

Samproxy is built by [CircleCI](https://circleci.com/gh/honeycombio/samproxy). Released versions of samproxy are available via Github under the Releases tab.

## Configuration

Configuration is done in one of two ways - entirely by the config file or a combination of the config file and a Redis backend for managing the list of peers in the cluster. When using Redis, it only manages peers - everything else is managed by the config file.

There are a few vital configuration options; read through this list and make sure all the variables are set.

### File-based Config

- API Keys: Samproxy itself needs to be configured with a list of your API keys. This lets it respond with a 401/Unauthorized if an unexpected api key is used. You can configure Samproxy to accept all API keys by setting it to `*` but then you will lose the authentication feedback to your application. Samproxy will accept all events even if those events will eventually be rejected by the Honeycomb API due to an API key issue.

- Goal Sample Rate and a list of fields to use to generate the keys off which sample rate is chosen. This is where the power of the proxy comes in - being able to dynamically choose sample rates based on the contents of the traces as they go by. There is an overall default and dataset-specific sections for this configuration, so that different datasets can have different sets of fields and goal sample rates.

- Trace timeout - it should be set higher (maybe double?) the longest expected trace. If all of your traces complete in under 10 seconds, 30 is a good value here.  If you have traces that can last minutes, it should be raised accordingly. Note that the trace doesn't *have* to complete before this timer expires - but the sampling decision will be made at that time. So any spans that contain fields that you want to use to compute the sample rate should arrive before this timer expires. Additional spans that arrive after the timer has expired will be sent or dropped according to the sampling decision made when the timer expired.

- Peer list: this is a list of all the other servers participating in this Samproxy cluster. Traces are evenly distributed across all available servers, and any one trace must be concentrated on one server, regardless of which server handled the incoming spans. The peer list lets the cluster move spans around to the server that is handling the trace. (Not used in the Redis-based config.)

- Buffer size: The `InMemCollector`'s `CacheCapacity` setting determines how many in-flight traces you can have. This should be large enough to avoid overflow. Some multiple (2x, 3x) the total number of in-flight traces you expect is a good place to start. If it's too low you will see the `collect_cache_buffer_overrun` metric increment. If you see that, you should increase the size of the buffer.

There are a few components of Samproxy with multiple implementations; the config file lets you choose which you'd like. As an example, there are two logging implementations - one that uses `logrus` and sends logs to STDOUT and a `honeycomb` implementation that sends the log messages to a Honeycomb dataset instead. Components with multiple implementations have one top level config item that lets you choose which implementation to use and then a section further down with additional config options for that choice (for example, the Honeycomb logger requires an API key).

When configuration changes, send Samproxy a USR1 signal and it will re-read the configuration.

### Redis-based Config

In the Redis-based config mode, all config options _except_ peer management are still handled by the config file.  Only coordinating the list of peers in the samproxy cluster is managed with Redis.

Enabling the redis-based config happens in one of two ways:
* set the `SAMPROXY_REDIS_HOST` environment variable
* use the flag `-p` or `--peer_type` with the argument `redis`

When launched in redis-config mode, Samproxy needs a redis host to use for managing the list of peers in the samproxy cluster. This hostname and port can be specified in one of two ways:
* set the `SAMPROXY_REDIS_HOST` environment variable
* set the `RedisHost` field in the config file

In other words, if you set the `SAMPROXY_REDIS_HOST` environment variable to the location of your redis host, you are done. Otherwise, launching samproxy with `-p redis` and setting the `RedisHost` field in the config file will accomplish the same thing.

The redis host should be a hostname and a port, for example `redis.mydomain.com:6379`. The example config file has `localhost:6379` which obviously will not work with more than one host.

## How sampling decisions are made

In the configuration file, there is a place to choose a sampling method and some options for each. The `DynamicSampler` is the most interesting and most commonly used, so that's the one that gets described here. It uses the `AvgSampleRate` algorithm from the [`dynsampler-go`](https://github.com/honeycombio/dynsampler-go) package. Briefly described, you configure samproxy to examine the trace for a set of fields (for example, `request.status_code` and `request.method`). It collects all the values found in those fields anywhere in the trace (eg "200" and "GET") together in to a key it hands to the dynsampler. The dynsampler code will look at the frequency that key appears during the previous 30 seconds (or other value set by the `ClearFrequencySec` setting) and use that to hand back a desired sample rate. More frequent keys are sampled more heavily, so that an even distribution of traffic across the keyspace is represented in Honeycomb.

By selecting fields well, you can drop significant amounts of traffic while still retaining good visibility into the areas of traffic that interest you. For example, if you want to make sure you have a complete list of all URL handlers invoked, you would add the URL (or a normalized form) as one of the fields to include. Be careful in your selection though, because if the combination of fields cretes a unique key each time, you won't sample out any traffic. Because of this it is not effective to use fields that have unique values (like a UUID) as one of the sampling fields. Each field included should ideally have values that appear many times within any given 30 second window in order to effectively turn in to a sample rate.

For more detail on how this algorithm works, please refer to the `dynsampler` package itself.

## Scaling Up

Samproxy uses bounded queues and circular buffers to manage allocating traces, so even under high volume memory use shouldn't expand dramatically. However, given that traces are stored in a circular buffer, when the throughput of traces exceeds the size of the buffer, things will start to go wrong. If you have stastics configured, a counter named `collect_cache_buffer_overrun` will be incremented each time this happens. The symptoms of this will be that traces will stop getting accumulated together, and instead spans that should be part of the same trace will be treated as two separate traces.  All traces will continue to be sent (and sampled) but the sampling decisions will be inconsistent so you'll wind up with partial traces making it through the sampler and it will be very confusing.  The size of the circular buffer is a configuration option named `CacheCapacity`. To choose a good value, you should consider the throughput of traces (eg traces / second started) and multiply that by the maximum duration of a trace (say, 3 seconds), then multiply that by some large buffer (maybe 10x). This will give you good headroom.

Determining the number of machines necessary in the cluster is not an exact science, and is best influenced by watching for buffer overruns. But for a rough heuristic, count on a single machine using about 2G of memory to handle 5000 incoming events and tracking 500 sub-second traces per second (for each full trace lasting less than a second and an average size of 10 spans per trace).

## Understanding Regular Operation

Samproxy emits a number of metrics to give some indication about the health of the process. These metrics can be exposed to Prometheus or sent up to Honeycomb. The interesting ones to watch are:

- Sample rates: how many traces are kept / dropped, and what does the sample rate distribution look like?
- [incoming|peer]_router_*: how many events (no trace info) vs. spans (have trace info) have been accepted, and how many sent on to peers?
- collect_cache_buffer_overrun: this should remain zero; a positive value indicates the need to grow the size of the collector's circular buffer (via configuration `CacheCapacity`).
- process_uptime_seconds: records the uptime of each process; look for unexpected restarts as a key towards memory constraints.

## Troubleshooting

The default logging level of `warn` is almost entirely silent. The `debug` level emits too much data to be used in production, but contains excellent information in a pre-production enviromnent. Setting the logging level to `debug` during initial configuration will help understand what's working and what's not, but when traffic volumes increase it should be set to `warn`.

## Restarts

Samproxy does not yet buffer traces or sampling decisions to disk. When you restart the process all in-flight traces will be flushed (sent upstream to Honeycomb), but you will lose the record of past trace decisions. When started back up, it will start with a clean slate.

## Architecture of Samproxy itself (for contributors)

Code segmentation

Within each directory, the interface the dependency exports is in the file with the same name as the directory and then (for the most part) each of the other files are alternative implementations of that interface.  For example, in `logger`, `/logger/logger.go` contains the interface definition and `logger/honeycomb.go` contains the implementation of the `logger` interface that will send logs to Honeycomb.

`main.go` sets up the app and makes choices about which versions of dependency implementations to use (eg which logger, which sampler, etc.) It starts up everything and then launches `App`

`app/app.go` is the main control point. When its `Start` function ends, the program shuts down. It launches two `Router`s which listen for incoming events.

`route/route.go` listens on the network for incoming traffic. There are two routers running and they handle different types of incoming traffic: events coming from the outside world (the `incoming` router) and events coming from another member of the samproxy cluster (`peer` traffic). Once it gets an event, it decides where it should go next: is this incoming request an event (or batch of events), and if so, does it have a trace ID? Everything that is not an event or an event that does not have a trace ID is immediately handed to `transmission` to be forwarded on to Honeycomb. If it is an event, the router extracts the trace ID and then uses the `sharder` to decide which member of the Samproxy cluster should handle this trace. If it's a peer, the event will be forwarded to that peer. If it's us, the event will be transformed in to an internal representation and handed to the `collector` to bundle up spans in to traces.

`collect/collect.go` the collector is responsible for bundling spans together in to traces and deciding when to send them to Honeycomb or if they should be dropped. The first time a trace ID is seen, the collector starts a timer. When that timer expires, the trace will be sent, whether or not it is complete. The arrival of the root span (aka a span with a trace ID and no parent ID) indicates the trace is complete. When that happens, the trace is sent and the timer canceled. Just before sending, the collector asks the `sampler` to give it a sample rate and whether to keep the trace. The collector obeys this sampling decision and records it (the record is applied to any spans that may come in as part of the trace after the decision has been made). After making the sampling decision, if the trace is to be kept, it is passed along to the `transmission` for actual sending.

`transmit/transmit.go` is a wrapper around the HTTP interactions with the Honeycomb API. It handles batching events together and sending them upstream.

`logger` and `metrics` are for managing the logs and metrics that Samproxy itself produces.

`sampler` contains algorithms to compute sample rates based on the traces provided.

`sharder` determines which peer in a clustered Samproxy config is supposed to handle and individual trace.

`types` contains a few type definitions that are used to hand data in between packages.

