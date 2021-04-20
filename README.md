# cni-ipoib-turbosds
| Cheng Wu  email: greatdolphin@gmail.com
"Sorry. No time to finish, but the primitive part, the bash script in pkg, is ok and work perfectly. Welcome pull requests to finish."

## Introduce
* A cni plugin working in Mallenox ib network by ipoib.
* Be compliant with https://github.com/containernetworking/cni/blob/spec-v0.4.0/SPEC.md
* This plugin is written by bash shell script, but wrapped by golang to prevent mistakes from reckless modifications.
* The minimum network config should have one ib hca card for inner communication of cluster and one ethern card for outer communication of cluster.

## Notice
* Only verified on rhel/centos 8.0 (kernel 4.18) by now. iptables in centos 7.x (kernel 3.10) is too low to run this plugin.

## Built-in ipam of ipv4
* Default pod network: 172.24.0.0/13. The pod network can be changed to others you intend, and the only constraint is netmask should be betweem 9 and 15.
* Default ib device: ib0
* The primary ip of ib device(ib0) should be 172.31.x.x/13 which should be ready before deploying this cni plugin. If you change the config of pod network, the second sector value of the primary ip of ib device should be calculated with expression: $(( (${net2f}&((255>>(16-${subnet_mask_size}))<<(16-${subnet_mask_size})))|(((255<<(${subnet_mask_size}-8))&255)>>(${subnet_mask_size}-8)) )). For example, new podnetwork set to 172.200.0.0/14, net2f=200, subnet_mask_size=14, the second sector value of primary ip of ib0 should be 203, and the final ip should be 172.203.x.x/14:

        # subnet_mask_size=12
        # net2f=200
        # echo $(( (${net2f}&((255>>(16-${subnet_mask_size}))<<(16-${subnet_mask_size})))|(((255<<(${subnet_mask_size}-8))&255)>>(${subnet_mask_size}-8)) ))
        203
* The available ips to assign to pods $(( (2**(16-${subnet_mask_size})-1)*254 )). It should be 762 ips in 3 subnet of 172.200.${net3f}.1-254, 172.201.${net3f}.1-254 and 172.202.${net3f}.1-254. ${net3f} is the 3rd sector of pod ip which comes from primary ip of host commonly on ethern card, and is used to distinguish pods in different host.

        # echo $(( (2**(16-${subnet_mask_size})-1)*254 ))
        762
