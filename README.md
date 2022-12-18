# IPFS-Community
Alpha Entanglement codes on the IPFS network to improve the reliability


## Introduction
The work continues the exploration as [Snarl](https://dl.acm.org/doi/pdf/10.1145/3464298.3493397). Snarl implements Alpha Entanglement codes on [Swarm](https://www.ethswarm.org/) to improve both file reliability and storage overhead. In this work, we try to simulate in IPFS what has been done in Swarm. 

The main challenge we are facing is that IPFS does not send your uploaded file to other users, namely your file is only locally available if no one explicitly requests it. We still call IPFS a distributed file system because the provider record of your uploaded file gets distributed inside the network, such that other nodes could find this file. However, because of the design, it becomes meaningless to perform neither replication, entanglement, or erasure codes on the data. After your local nodes go offline, no one will be able to use these redundancies to retrieve/recover the data.

We need more mechanisms than IPFS alone to achieve the goal that we have. In this project, we propose the usage of [IPFS Cluster](https://github.com/ipfs-cluster/ipfs-cluster) to distribute files and make them remotely available. There are some limitations with IPFS Cluster, for example, it runs on top of IPFS, which means you have to explicitly set it up to use its service. Moreover, it is only privately connected, i.e., you have to know the cluster secret and the bootstrap nodes to join the cluster. The design of the IPFS Cluster defines the scope of this project. We are considering the following scenarios:
1. A small group of friends/families/colleagues are trying to hold some files, such that they would be able to access all these files at all times.
2. A large group of strangers tried to form a community, where each other tries to help each other to make their file always available. This case might be a bit unrealistic, since the security guarantee of the IPFS Cluster is weak, and there is no incentive for each individual to provide this service. This is the ultimate goal of this project, the future works of this project might involve some aspects of trying to solve these issues.


## To run the program

### Prerequisite
You have to have at least one IPFS node running on your computer (either from IPFS daemon, docker, or ...), which allows you to upload and download from the IPFS network. You could also set up an IPFS Cluster, using the `docker-compose.yml` in the directory. You could change the number of cluster peers you want to support inside the file.

### Commands

To uploade files with entanglement (alpha = 3, s = 5, p = 5):
```
go run main.go upload <path_to_file> --alpha 3 -s 5 -p 5
```

To download files with recovery enable:
```
go run main.go download <file_CID> -o <output_path> -m <metadata_CID> -u <enable_missing_block_upload>
```

To do performance test:
```
go run main.go perf recover -t <test_case> -p <loss_percent_of_parities> -i <iteration>
go run main.go perf rep -t <test_case> -p <loss_percent_of_replication> -i <iteration> -r <replication_factor>
```
