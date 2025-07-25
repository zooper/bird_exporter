# Prefix Size Statistics Example

This example shows how to use the new prefix size statistics feature to track the distribution of route prefix lengths.

## Enable Prefix Size Collection

```bash
# Start bird_exporter with prefix size statistics enabled
./bird_exporter -prefix.size=true
```

## Example Metrics Output

When enabled, you'll see metrics like these in the `/metrics` endpoint:

```
# HELP bird_prefix_length_count Number of prefixes by prefix length
# TYPE bird_prefix_length_count gauge
bird_prefix_length_count{name="bgp1",proto="BGP",ip_version="4",prefix_length="8"} 1
bird_prefix_length_count{name="bgp1",proto="BGP",ip_version="4",prefix_length="16"} 5
bird_prefix_length_count{name="bgp1",proto="BGP",ip_version="4",prefix_length="24"} 150
bird_prefix_length_count{name="bgp1",proto="BGP",ip_version="4",prefix_length="32"} 10
bird_prefix_length_count{name="bgp1",proto="BGP",ip_version="6",prefix_length="32"} 2
bird_prefix_length_count{name="bgp1",proto="BGP",ip_version="6",prefix_length="48"} 25
bird_prefix_length_count{name="bgp1",proto="BGP",ip_version="6",prefix_length="64"} 100
```

## Supported Protocols

Prefix size statistics are collected for routing protocols that maintain route tables:
- BGP
- OSPF  
- Kernel
- Static
- Direct
- Babel

## BIRD Requirements

The feature requires:
- BIRD routing daemon running and accessible via socket
- Proper permissions to query `show route protocol {name}` command
- BIRD configured with the protocols you want to monitor

## Performance Considerations

- Enable only when needed (`-prefix.size=false` by default)
- Queries each protocol's routing table individually
- May impact performance on systems with large routing tables
- Consider scrape frequency in Prometheus configuration

## Example Prometheus Queries

```promql
# Total prefixes by length across all protocols
sum by (prefix_length) (bird_prefix_length_count)

# IPv4 /24 prefixes by protocol
bird_prefix_length_count{ip_version="4",prefix_length="24"}

# BGP prefix distribution
bird_prefix_length_count{proto="BGP"}
```