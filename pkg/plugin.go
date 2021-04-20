package pkg

import (
	"github.com/ChengWu-NJ/golangutils/utils"
)

func RunPlugin() (*utils.BashOutput, error) {
	return utils.LocalBashRun(BashScript, 5)
}

var BashScript string = `
#!/bin/bash -e

if [[ ${DEBUG} -gt 0 ]]; then set -x; fi

# make stdout available as fd 3 for the result
exec 3>&1
exec &>> /var/log/cni-ipoib-ts.log

log(){
  echo "$(date --iso-8601=ns) $1"
}

getvaluebykey() {
  local instr=${1//\{/}
  local key=$2

  echo ${instr//\}/}
  outstr=$(echo ${instr//\}/}| awk -F: '$1=="\"'${key}'\""{print $2}' RS=',')
  echo ${outstr//\"/}
}

allocate_ip(){
  reserved_ips=$(cat ${IP_STORE} 2> /dev/null)
  reserved_ips=(${reserved_ips})
  for ip in "${all_ips[@]}"
  do
    reserved=false
    for reserved_ip in "${reserved_ips[@]}"
    do
      if [ "${ip}" = "${reserved_ip}" ]; then
        reserved=true
        break
      fi
    done
    if [ "${reserved}" = false ] ; then
      echo "${ip}" >> ${IP_STORE}
      sync ${IP_STORE}
      echo "${ip}"
      return
    fi
  done
}

linknetns(){
  mkdir -p /var/run/netns/
  ln -sfT ${CNI_NETNS} /var/run/netns/${CNI_CONTAINERID}
}

checkiptables(){
  if [[ -z $(iptables -t mangle -L FORWARD |grep cni-ipoib-ts-mangle-forward) ]];then
    iptables -t mangle -A FORWARD ! -s 127.0.0.0/8 -m comment --comment "cni-ipoib-ts-mangle-forward" -j ACCEPT
  fi
  if [[ -z $(iptables -t filter -L FORWARD |grep cni-ipoib-ts-filter-forward) ]];then
    iptables -t filter -A FORWARD -s ${network} -m comment --comment "cni-ipoib-ts-filter-forward" -j ACCEPT
  fi
  if [[ -z $(iptables -t nat -L POSTROUTING |grep cni-ipoib-ts-nat-masquerade) ]];then
    iptables -t nat -A POSTROUTING -s ${network} -m comment --comment "cni-ipoib-ts-nat-masquerade" -j MASQUERADE
  fi
}

RUNDIR="/var/run/cni-ipoib-ts"
mkdir -p ${RUNDIR}
# already-assigned ips will be stored there
IP_STORE="${RUNDIR}/ip4net_reserved_ips"
# the first interface name must be eth0 according to the conventional codes
THE_ONLY_ONE_CNI_DEV="eth0"

log "CNI command: ${CNI_COMMAND}"

stdin=$(cat /dev/stdin)
log "stdin: ${stdin}"

case ${CNI_COMMAND} in
ADD)
  exec 9>${RUNDIR}/assignIP.lck || exit 9
  flock --timeout=3 9 || exit 9

  network=$(getvaluebykey "$stdin" "network")
  ### check iptables rule
  $(checkiptables)

  ### a simple hash algorithm: using the 4th section of main ip to be the 3rd of subnet
  net1f=$(echo ${network}|cut -d'.' -f1)
  net2f=$(echo ${network}|cut -d'.' -f2)
  net3f=$(hostname -i|cut -d' ' -f1|cut -d'.' -f4)
  subnet_mask_size=$(echo ${network} | awk -F  "/" '{print $2}')
  if [[ ${subnet_mask_size} -gt 15 || ${subnet_mask_size} -lt 9 ]];then
    log "[ERROR]: subnet mask length ${subnet_mask_size} is wrong which should between 9 and 15"
	exit 2
  fi
  ### -1 means the ib dev primary ip, should be in the max subnet
  nr_subnets=$(( 2**(16-${subnet_mask_size})-1 ))
  
  ibdev=$(getvaluebykey "$stdin" "ibdev")
  ip addr show ${ibdev} > /dev/null || log "[ERROR]: ${ibdev} dose not exist"; exit 3
  ibip=$(ip addr show ${ibdev}|grep -E "inet.*global ${ibdev}"|awk '{print $2}'|sed  s%%/.*%%%%)

  #certainly, ip of ib should be 172.xx.yy.zz
  ibip1f=$(echo ${ibip}|cut -d'.' -f1)
  ibip2f=$(echo ${ibip}|cut -d'.' -f2)
  if [[ ( ! ${ibip1f} =~ ^[0-9]+$ ) || ( ! ${ibip2f} =~ ^[0-9]+$ ) ]]; then log "failed to get ip of ${ibdev}"; exit 4; fi
  right2f=$(( (${net2f}&((255>>(16-${subnet_mask_size}))<<(16-${subnet_mask_size})))|(((255<<(${subnet_mask_size}-8))&255)>>(${subnet_mask_size}-8)) ))
  if [[ ( ${ibip1f} != ${net1f} ) || ( ${ibip2f} != ${right2f} ) ]];then
    log "the primary ip of ${ibdev} should ${net1f}.${right2f}.x.x/${subnet_mask_size}"
	exit 5
  fi

  for _ in $(seq ${nr_subnets});do
    ### skip .0 and .255
    all_ips=$(for idx in {1..254};do echo ${net1f}.${net2f}.${net3f}.${idx};done)
    all_ips=(${all_ips[@]})
    container_ip=$(allocate_ip)
	if [[ ! -z ${container_ip} ]];then
	  break
	fi
	net2f=$(( ${net2f}+1 ))
  done


  $(linknetns)

  # random to avoid duplicated name
  rand=$(tr -dc 'A-F0-9' < /dev/urandom | head -c4)
  cni_ipoib_ts_dev_name="${ibdev}_${rand}"
  ip link add link ${ibdev} name ${cni_ipoib_ts_dev_name} type ipoib
  ip link set netns ${CNI_CONTAINERID} dev ${cni_ipoib_ts_dev_name}
  ip netns exec ${CNI_CONTAINERID} ip link set ${cni_ipoib_ts_dev_name} name ${THE_ONLY_ONE_CNI_DEV}
  ip netns exec ${CNI_CONTAINERID} ip addr add ${container_ip}/${subnet_mask_size} dev ${THE_ONLY_ONE_CNI_DEV}
  ip netns exec ${CNI_CONTAINERID} ip link set ${THE_ONLY_ONE_CNI_DEV} up
  ip netns exec ${CNI_CONTAINERID} ip route add default via ${ibip} dev ${THE_ONLY_ONE_CNI_DEV}
  default_gateway=$(ip netns exec ${CNI_CONTAINERID} ip route show | grep "default via"|cut -d' ' -f3)
  mac=$(ip netns exec ${CNI_CONTAINERID} ip link show ${THE_ONLY_ONE_CNI_DEV} | awk '/infiniband/ {print $2}')

  addresult="{
  \"cniVersion\": \"0.3.1\",
  \"interfaces\": [
      {
          \"name\": \"${THE_ONLY_ONE_CNI_DEV}\",
          \"mac\": \"${mac}\",
          \"sandbox\": \"${CNI_NETNS}\"
      }
  ],
  \"ips\": [
      {
          \"version\": \"4\",
          \"address\": \"${container_ip}/${subnet_mask_size}\"
      }
  ],
  \"default-gateway\": \"${default_gateway}\"
}"
  log "ADD-RESULT: ${addresult}"
        echo ${addresult}>&3

;;

DEL)
  if [ -f /var/run/netns/${CNI_CONTAINERID} ];then
    $(linknetns)
    ips=$(ip netns exec ${CNI_CONTAINERID} ip addr show|awk '/inet / {print $2}' | sed  s%%/.*%%%%)
    ips=(${ips})

    for ip in "${ips[@]}"
    do
      if [ ! -z "$ip" ]
      then
        sed -i "/$ip/d" $IP_STORE
      fi
    done

    ip netns exec ${CNI_CONTAINERID} ip link delete ${THE_ONLY_ONE_CNI_DEV} 2>/dev/null
    rm -f /var/run/netns/${CNI_CONTAINERID}
  fi
;;

GET)
  log "GET not supported"
  exit 1
;;

VERSION)
echo '{
  "cniVersion": "0.4.0",
  "supportedVersions": [ "0.3.0", "0.3.1", "0.4.0" ]
}' >&3
;;

*)
  log "Unknown cni command: ${CNI_COMMAND}"
  exit 1
;;

esac
`
