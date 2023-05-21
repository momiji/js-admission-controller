package main

const (
	GroupCrd   = "momiji.com"
	VersionCrd = "v1"

	ClusterCrd   = GroupCrd + "/" + VersionCrd + "/clusterjsadmissions"
	NamespaceCrd = GroupCrd + "/" + VersionCrd + "/jsadmissions"

	Mutate   = "jsa_mutate"
	Validate = "jsa_validate"
)
