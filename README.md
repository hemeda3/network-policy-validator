# Learning GO & K8, by developing  Kubernates Network Policy Validator

*1*- extract network policy
*2*- for each network policy  do :
  inside for each ingress/egress do  :
    get allowed pod , ports, protocols 
4- generate tests cases based on the information 
5- use port scanner tool to execute test case
6- gnerate reports
7- support Calico ( including Calico custom policy), Weave net, etc ...    


* skeleton code copied from tgik-controller/Vmware 
