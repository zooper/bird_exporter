# Adding BGP Prefix Length Statistics to BIRD Exporter

*Posted: 2025-01-25 | 5 minute read*

When you're running BGP full tables (like the ~220k IPv6 routes I receive), understanding prefix length distribution becomes interesting for network analysis. The question is simple: how many /48s, /32s, /44s am I actually receiving?

The popular [bird_exporter](https://github.com/czerwonk/bird_exporter) for Prometheus monitoring didn't support this out of the box. Time to fix that.

## The Problem

BIRD exporter gives you great metrics about BGP session states, route counts, and protocol health. But it doesn't break down prefix lengths - essentially treating all routes equally. For understanding your BGP table composition, this granular data is interesting.

Looking at my setup receiving IPv6 full tables, I wanted to see something like:
- /48 prefixes: 197,997 routes (customer allocations)  
- /32 prefixes: 51,502 routes (ISP allocations)
- /44 prefixes: 41,519 routes
- And so on...

## The Naive Approach (That Doesn't Work)

My first instinct was straightforward: query all routes and parse prefix lengths.

```bash
show route table master6
```

This works fine for small routing tables. But with 220k routes, each generating ~4 lines of BIRD output (primary route, alternatives, interface info), you're looking at 880k lines of data.

Result: Unix socket buffer overflow. Truncated data. Only ~3,300 routes parsed instead of 220k.

## Socket Buffer Reality Check

The core issue became clear quickly. BIRD's Unix socket has buffer limits, and massive route dumps exceed them. Even chunking by prefix length didn't help - a single `/48` query returned 200k+ lines, still too large.

This is where many similar projects probably gave up. But BIRD has a better way.

## The Solution: Count Commands

Instead of parsing full route details, BIRD v2 supports efficient count queries:

```bash
show route table master6 where net ~ [::/0{48,48}] count
```

This returns just:
```
1007-197991 of 220785 routes for 220785 networks in table master6
```

Perfect! The number after the dash (197991) is exactly what we need - the count of /48 prefixes.

## Implementation Details

The implementation iterates through common prefix lengths and queries each:

```go
// IPv6 common prefix lengths
prefixLengths := []int{20, 24, 28, 29, 30, 32, 36, 40, 44, 48, 52, 56, 60, 64, 96, 128}

for _, prefixLen := range prefixLengths {
    cmd := fmt.Sprintf("show route table %s where net ~ [::/0{%d,%d}] count", 
                      tableName, prefixLen, prefixLen)
    // Query and parse count...
}
```

The key insight: each count query returns one small response instead of hundreds of thousands of route lines.

## Parsing BIRD v2 Output

BIRD v2's count format needed special handling:
```
1007-197991 of 220785 routes for 220785 networks in table master6
```

Where:
- `1007` = response code
- `197991` = filtered route count (what we want)
- `220785` = total routes in table

A simple regex extracts the middle number:
```go
countRegex := regexp.MustCompile(`^(\d+)-(\d+)\s+of\s+\d+\s+routes`)
```

## Results

The implementation now exports clean Prometheus metrics:

```
# HELP bird_table_prefix_length_count Number of unique prefixes by prefix length in routing table
# TYPE bird_table_prefix_length_count gauge
bird_table_prefix_length_count{ip_version="6",prefix_length="48",table="master6"} 198002
bird_table_prefix_length_count{ip_version="6",prefix_length="32",table="master6"} 51500
bird_table_prefix_length_count{ip_version="6",prefix_length="40",table="master6"} 46308
bird_table_prefix_length_count{ip_version="6",prefix_length="44",table="master6"} 41514
bird_table_prefix_length_count{ip_version="6",prefix_length="36",table="master6"} 17230
bird_table_prefix_length_count{ip_version="6",prefix_length="29",table="master6"} 10321
bird_table_prefix_length_count{ip_version="6",prefix_length="30",table="master6"} 1478
bird_table_prefix_length_count{ip_version="6",prefix_length="28",table="master6"} 308
bird_table_prefix_length_count{ip_version="6",prefix_length="24",table="master6"} 74
bird_table_prefix_length_count{ip_version="6",prefix_length="20",table="master6"} 24
```

Perfect for Grafana dashboards showing prefix length distribution over time.

## Usage

The feature is enabled with a simple flag:

```bash
./bird_exporter -bird.v2 -bird.socket /var/run/bird/bird.ctl -prefix.size.table=true
```

It automatically detects both IPv4 and IPv6 BGP tables, generating separate metrics for each IP version.

## Technical Lessons

1. **Don't parse what you can count** - BIRD's count commands are far more efficient than full route parsing
2. **Socket buffers have limits** - Large datasets require different approaches than small ones  
3. **Read the manual** - BIRD v2's filter syntax and count capabilities weren't immediately obvious
4. **Test with real data** - Small test datasets hide scalability issues

## The Code

The implementation is available on my [GitHub fork](https://github.com/zooper/bird_exporter) with the complete table-wide prefix statistics feature.

Now I can finally answer the question: "What's actually in my BGP table?" with proper metrics and monitoring.

*Tagged: BGP, BIRD, Prometheus, IPv6, Network Monitoring*