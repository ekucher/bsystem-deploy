# Remote Management Network

## Topology

Use a central hub-and-spoke management network. An engineer connects to one corporate WireGuard tunnel and reaches only devices authorized by policy.

Working overlay address space:

```text
10.253.0.0/16
```

This range requires a conflict audit before production deployment.

Example allocation:

```text
10.253.0.1     WG Hub
10.253.0.5     Zabbix
10.253.0.6     Provisioning API
10.253.0.10    engineer-a
10.253.10.17   Customer A / WIN2016-LIMS
10.253.11.20   Customer B / SERVER01
```

Do not base remote management on customer LAN addresses. Multiple customers may use the same RFC1918 subnets.

## WireGuard identities

Each direct managed device has a unique private/public key pair and unique management address.

Each engineer has a unique peer. Shared `support.conf` is prohibited.

Device private keys are generated locally and do not leave the device. Backend receives the public key and supplies the management address, hub public key, endpoint, required routes and keepalive policy.

## Network authorization

WireGuard `AllowedIPs` is transport/routing configuration, not the sole RBAC control.

Central hub firewall/ACL enforcement must be deny-by-default and derived from authorized desired state.

Required invariant:

```text
Customer A device -> Customer B device = DENY
```

Customer devices also must not gain implicit access to BSYSTEM internal LAN, engineer workstations, databases or administrative provisioning endpoints.

## RDP

RDP is reachable only through authorized management paths.

```text
Internet -> TCP/3389 = DENY
Management policy -> device:3389 = ALLOW when authorized
NLA = REQUIRED
```

## Zabbix

Zabbix uses the same management overlay. Prefer Zabbix Agent 2 active checks for remote managed servers.

Do not expose customer TCP/10050 to WAN.

Central health should also evaluate WG handshake, management reachability, TCP/3389, Zabbix freshness and RustDesk state rather than trusting only endpoint self-reporting.

## Public ports

Managed customer Windows Servers require no inbound public port forwarding for normal operation.

Customer side:

```text
WAN -> 3389                 DENY
WAN -> 10050                DENY
WAN -> Windows WG listener  NOT REQUIRED
WAN -> RustDesk client      NOT REQUIRED
```

BSYSTEM side requires a public WireGuard endpoint, normally UDP/51820.

Self-hosted RustDesk infrastructure may require its server-side ports according to the selected deployment; web/API/console should use HTTPS/443 where supported.

## Legacy transport

Windows Server 2012 R2 must not force the whole model to depend on a current WireGuard client.

Supported design includes `wireguard-gateway` and `openvpn` for legacy cases. A gateway-managed device may reference `gatewayId`, internal address and management route instead of owning a WG peer.
