# hcloud-ip-failover: Floating and alias IP address failover on Hetzner Cloud

This simple Go client is intended for scenarios where two identical servers on Hetzner Cloud need to be able to failover both a floating IP address and an alias IP address. This client enables a crude failover of e.g. two pfSense systems in a high availability configuration, as CARP does not function properly on L3 networks such as those of Hetzner Cloud.

## Installation

Perform the following action on the *primary* system:

1. Add the address of the primary `SYNC` interface as a gateway, and configure the alert threshold as desired. This will make the secondary system trigger its `/etc/rc.gateway_alarm` when the primary system is down or back up.

Perform the following actions on the *secondary* (backup) system:

1. Run the following commands to download a release, extract it to `/root` (or some other directory) and ensure it is executable:

    curl -sSL https://github.com/mikavl/hcloud-ip-failover/releases/download/v0.0.1/hcloud-ip-failover-v0.0.1-freebsd-amd64.tar.gz | tar -xzC /root
    chown root:root /root/hcloud-ip-failover
    chmod 0755 /root/hcloud-ip-failover

2. Create a Hetzner Cloud token and place it in `/root/.hcloud_token`, or place it somewhere else and add the `--token-path` argument to the binary in the next step.

3. Add a call to `/root/hcloud-ip-failover` to `/etc/rc.gateway_alarm` in order to trigger the failover action when the primary system stops responding on the `SYNC` interface or comes back up:

    # Change the gateway name to whatever you configured on the primary system
    if [ "xPRIMARY_SYNC_GW" = "x$GW" ]; then
      /root/hcloud-ip-failover "$alarm_flag" > /dev/null 2>&1
    fi

4. Test! Shut down the primary, or block ICMP ping on the `SYNC` interface, and ensure that the floating and alias IPs get assigned to the secondary as intended. Bring the primary up again and see that the IPs get assigned back to it.
