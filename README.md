#Enforcer


### Overview
Enforcer acts as a webserver for ipxe configurations, a bootserver if you will. By default it has support for dockers linked containers which meaning its able to run dockerized by default. Data is stored in RethinkDB which means that we can put all parts in a container. Enforcer has a builtin client for Serf so i t can retrieve members and information from these.

#### How it should work.

An ipxe is prepared for provisioning via tftp.
Enforcer sets up a communication to an agent, for example serf or consul and activates the agents ipxe.
The client machine is beeing booting requesting the prepared general ipxe.
The client machine request locally the bootserver for the ipxe for its macaddress, if unsuccessful the agents ipxe and if no success go to a remote instance of Enforcer. If no success it boots to the local harddrive.
Booting to an agent ipxe adds a communication channel between enforcer and agent. Enforcer will retrieve data such as mac address and any information that can identify the hardware in any way.
From the information we define a new ipxe conf for that specific machine. Note that Enforcer should retrieve most of the information. You should only need to specify the os, location of files and a script to trigger the provisioning.
After ipxe is created you active the configuration and send a reboot command to the client machine making it trigger your ipxe conf.
After the client machine has triggered it will make the ipxe conf unavailable. Enforcer works in an environment where it expects tha machines always boot from pxe.

#### How it works.

So far we can set up ipxe configurations manually. Setting up connections to agents and retrieving member list is also possible.


#### Tomorrow

The new version of Enforcer drops the database and its driver as a dependency. There will be support for multiple agents that is able to speak a structurized language for communication such as json. Enforcer will have an api and the webapp will be a seperate client communicating over that api. It now focuses on working completely on small devices with raspberry pi to start of with. Now, RethinkDB has fully support for the pi but due to the resource usage we can drop by removing the database its worth it.
