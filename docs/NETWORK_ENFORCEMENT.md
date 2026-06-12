# Network enforcement

## Client firewall policy

Canonical script:

```text
client/firewall/firewall.sh
```

Policy:

- `iptables` INPUT/OUTPUT/FORWARD default DROP;
- `ip6tables` INPUT/OUTPUT/FORWARD default DROP;
- allow loopback;
- allow established/related;
- redirect non-Tor TCP to Tor `TransPort`;
- redirect DNS UDP to Tor `DNSPort`;
- allow Tor daemon output;
- drop other UDP;
- drop ICMP.

## Tor client config

```text
TransPort 9040
DNSPort 5353
AutomapHostsOnResolve 1
VirtualAddrNetworkIPv4 10.192.0.0/10
DataDirectory /run/tor
Log notice file /run/tor/tor.log
SocksPort 9050
```

`/run/tor` is tmpfs-backed.

## IPv6 disablement

```text
net.ipv6.conf.all.disable_ipv6 = 1
net.ipv6.conf.default.disable_ipv6 = 1
net.ipv6.conf.lo.disable_ipv6 = 1
```

## Service ordering

`deaddrop-firewall.service` uses:

```ini
DefaultDependencies=no
Before=network-pre.target
WantedBy=network-pre.target
```

If firewall application fails, networking must not proceed.
